package ar.com.tengolibre.app.nucleo

import android.app.Activity
import android.content.Context
import ar.com.tengolibre.app.app.Estado
import com.android.billingclient.api.AcknowledgePurchaseParams
import com.android.billingclient.api.BillingClient
import com.android.billingclient.api.BillingClientStateListener
import com.android.billingclient.api.BillingFlowParams
import com.android.billingclient.api.BillingResult
import com.android.billingclient.api.PendingPurchasesParams
import com.android.billingclient.api.ProductDetails
import com.android.billingclient.api.Purchase
import com.android.billingclient.api.QueryProductDetailsParams
import com.android.billingclient.api.QueryPurchasesParams
import com.android.billingclient.api.acknowledgePurchase
import com.android.billingclient.api.queryProductDetails
import com.android.billingclient.api.queryPurchasesAsync
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable

// Clon de Nucleo/Compras.swift con Play Billing.

/** La suscripción desde el teléfono (Play Billing): los planes son suscripciones mensuales de Google Play con los
 *  mismos ids que en la App Store. Cada compra lleva el id del negocio (obfuscatedAccountId) y su token va a la API
 *  (`POST /v1/panel/suscripcion/google`), que además recibe las renovaciones y cancelaciones por RTDN. Desde la web
 *  se sigue pagando con Mercado Pago. */
object Compras {
    val productos = mapOf(
        "ar.com.tengolibre.inicial.mensual" to "inicial",
        "ar.com.tengolibre.completo.mensual" to "completo",
    )

    enum class Resultado { listo, cancelado, pendiente }

    /** Un plan listo para mostrar: el detalle de Play, su precio formateado y la oferta a comprar. */
    data class Plan(val id: String, val nombre: String, val descripcion: String, val precio: String, val micros: Long, val oferta: String, val detalle: ProductDetails)

    private var cliente: BillingClient? = null
    private var esperando: CompletableDeferred<Pair<BillingResult, List<Purchase>?>>? = null
    private var estado: Estado? = null
    private val alcance = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    private suspend fun conectado(ctx: Context): BillingClient? {
        val c = cliente ?: BillingClient.newBuilder(ctx.applicationContext)
            .setListener { r, compras ->
                val e = esperando
                if (e != null && !e.isCompleted) e.complete(r to compras)
                // Compras que llegan solas (pendientes que se aprobaron, otra cuenta): se registran igual.
                else if (r.responseCode == BillingClient.BillingResponseCode.OK) estado?.let { est -> alcance.launch { compras?.forEach { runCatching { registrar(it, est) } } } }
            }
            .enablePendingPurchases(PendingPurchasesParams.newBuilder().enableOneTimeProducts().build())
            .enableAutoServiceReconnection()
            .build().also { cliente = it }
        if (c.isReady) return c
        val listo = CompletableDeferred<Boolean>()
        c.startConnection(object : BillingClientStateListener {
            override fun onBillingSetupFinished(r: BillingResult) { listo.complete(r.responseCode == BillingClient.BillingResponseCode.OK) }
            override fun onBillingServiceDisconnected() { if (!listo.isCompleted) listo.complete(false) }
        })
        return if (listo.await()) c else null
    }

    /** Los planes de Google Play, del más barato al más caro (vacío si la tienda no los da). */
    suspend fun planes(ctx: Context): List<Plan> {
        if (ModoPrueba.activo) return emptyList()
        val c = conectado(ctx) ?: return emptyList()
        val p = QueryProductDetailsParams.newBuilder().setProductList(productos.keys.map {
            QueryProductDetailsParams.Product.newBuilder().setProductId(it).setProductType(BillingClient.ProductType.SUBS).build()
        }).build()
        val r = runCatching { c.queryProductDetails(p) }.getOrNull() ?: return emptyList()
        return r.productDetailsList.orEmpty().mapNotNull { d ->
            val o = d.subscriptionOfferDetails?.firstOrNull() ?: return@mapNotNull null
            val fase = o.pricingPhases.pricingPhaseList.lastOrNull() ?: return@mapNotNull null
            Plan(d.productId, d.name, d.description, fase.formattedPrice, fase.priceAmountMicros, o.offerToken, d)
        }.sortedBy { it.micros }
    }

    /** Compra para el negocio elegido y le pasa el token a la API. */
    suspend fun comprar(actividad: Activity, p: Plan, estado: Estado): Resultado {
        val negocio = estado.negocio?.id ?: return Resultado.cancelado
        this.estado = estado
        val c = conectado(actividad) ?: throw ErrorApi.desconocido
        val e = CompletableDeferred<Pair<BillingResult, List<Purchase>?>>().also { esperando = it }
        val params = BillingFlowParams.newBuilder()
            .setProductDetailsParamsList(listOf(BillingFlowParams.ProductDetailsParams.newBuilder().setProductDetails(p.detalle).setOfferToken(p.oferta).build()))
            .setObfuscatedAccountId(negocio)
            .build()
        val inicio = c.launchBillingFlow(actividad, params)
        if (inicio.responseCode != BillingClient.BillingResponseCode.OK) { esperando = null; return Resultado.cancelado }
        val (r, compras) = e.await()
        esperando = null
        if (r.responseCode != BillingClient.BillingResponseCode.OK) return Resultado.cancelado
        val compra = compras?.firstOrNull { it.products.any { id -> id in productos } } ?: return Resultado.cancelado
        if (compra.purchaseState == Purchase.PurchaseState.PENDING) return Resultado.pendiente
        registrar(compra, estado)
        return Resultado.listo
    }

    @Serializable private data class Cuerpo(val producto: String, val token: String)

    /** Manda a la API una compra de Play y la reconoce (si no, Play la devuelve a los tres días). */
    suspend fun registrar(compra: Purchase, estado: Estado) {
        if (compra.purchaseState != Purchase.PurchaseState.PURCHASED) return
        val producto = compra.products.firstOrNull { it in productos } ?: return
        estado.api.hacer("POST", "/v1/panel/suscripcion/google", Cuerpo(producto, compra.purchaseToken))
        if (!compra.isAcknowledged) cliente?.acknowledgePurchase(AcknowledgePurchaseParams.newBuilder().setPurchaseToken(compra.purchaseToken).build())
        estado.version += 1
        estado.refrescar()
    }

    /** «Restaurar compras»: vuelve a mandar a la API las suscripciones vigentes de esta cuenta de Google. */
    suspend fun restaurar(ctx: Context, estado: Estado): Int {
        this.estado = estado
        val c = conectado(ctx) ?: return 0
        val r = c.queryPurchasesAsync(QueryPurchasesParams.newBuilder().setProductType(BillingClient.ProductType.SUBS).build())
        var n = 0
        r.purchasesList.filter { it.products.any { id -> id in productos } }.forEach { registrar(it, estado); n++ }
        return n
    }

    /** Al abrir la app: lo que quedó sin registrar (pendientes aprobadas, compras en otro teléfono). */
    fun escuchar(ctx: Context, estado: Estado) {
        this.estado = estado
        if (ModoPrueba.activo) return
        alcance.launch {
            runCatching {
                val c = conectado(ctx) ?: return@runCatching
                c.queryPurchasesAsync(QueryPurchasesParams.newBuilder().setProductType(BillingClient.ProductType.SUBS).build())
                    .purchasesList.filter { !it.isAcknowledged && it.products.any { id -> id in productos } }.forEach { registrar(it, estado) }
            }
        }
    }
}
