package ar.com.tengolibre.app

import android.Manifest
import android.content.ActivityNotFoundException
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.browser.customtabs.CustomTabsIntent
import androidx.compose.runtime.CompositionLocalProvider
import androidx.lifecycle.lifecycleScope
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.app.Raiz
import ar.com.tengolibre.app.app.Ruta
import ar.com.tengolibre.app.nucleo.Clerk
import ar.com.tengolibre.app.nucleo.Compras
import ar.com.tengolibre.app.nucleo.Guardado
import ar.com.tengolibre.app.nucleo.Llavero
import ar.com.tengolibre.app.nucleo.ModoPrueba
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.pantallas.mas.EstadoNegocio
import ar.com.tengolibre.app.push.EnPrimerPlano
import ar.com.tengolibre.app.push.Push
import ar.com.tengolibre.app.tema.Haptico
import ar.com.tengolibre.app.widgets.Widgets
import ar.com.tengolibre.app.tema.TemaTL
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.launch
import java.lang.ref.WeakReference

/** La única actividad (TengoLibreApp en iOS): arma el Estado, recibe los links, la vuelta de Clerk y el permiso de
 *  avisos, y prende o apaga el en vivo al entrar y salir. */
class MainActivity : ComponentActivity() {
    private lateinit var estado: Estado
    /** Si la Custom Tab del ingreso volvió con la vuelta de Clerk (si no, la persona la cerró: «cancelado» en iOS). */
    private var vueltaClerk = false
    private var permiso: CompletableDeferred<Boolean>? = null
    private val pedirPermiso = registerForActivityResult(ActivityResultContracts.RequestPermission()) { ok -> permiso?.complete(ok); permiso = null }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Preferencias.contexto = applicationContext
        Guardado.contexto = applicationContext
        Llavero.contexto = applicationContext
        if (BuildConfig.DEBUG && intent?.getBooleanExtra("modoPrueba", false) == true) {
            ModoPrueba.activo = true
            ModoPrueba.cliente = intent.getBooleanExtra("modoCliente", false)
            ModoPrueba.respuestas = { assets.open("pruebas/respuestas.json").bufferedReader().readText() }
            ModoPrueba.preparar(intent.getBooleanExtra("conRecorrido", false))
        }
        // La hora y los íconos de arriba siempre en blanco, sobre el petróleo de la cabecera (UIStatusBarStyleLightContent).
        enableEdgeToEdge(statusBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT))
        Haptico.vista = WeakReference(window.decorView)
        estado = Estado()
        estado.abrirPestana = { url -> vueltaClerk = false; CustomTabsIntent.Builder().setShowTitle(true).build().launchUrl(this, Uri.parse(url)) }
        estado.alPedirPermisoPush = { pedirPermisoAvisos() }
        estado.compartirTexto = { t -> startActivity(Intent.createChooser(Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, t), null)) }
        // tel:, whatsapp:, mailto:… En un equipo sin la app que corresponde (una tablet sin marcador) no se cae: avisa.
        estado.abrirLink = { u ->
            try {
                startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(u)))
            } catch (e: ActivityNotFoundException) {
                estado.avisar(if (u.startsWith("tel:")) "Este equipo no puede llamar" else "No hay una app para abrir esto", ok = false)
            } catch (e: Exception) {
                estado.avisar("No pudimos abrirlo", ok = false)
            }
        }
        // Push (FCM), los widgets y lo que mira «Potenciá tu negocio».
        Push.iniciar(applicationContext)
        estado.alPedirTokenPush = { Push.registrar(applicationContext, estado) }
        estado.alPedirAvisosClienta = {
            if (!Preferencias.bool("avisosClientaPedidos")) { Preferencias.ponerBool("avisosClientaPedidos", true); pedirPermisoAvisos() }
            Push.registrar(applicationContext, estado)
        }
        EstadoNegocio.hayAvisos = { Push.hayPermiso(applicationContext) }
        EstadoNegocio.hayWidget = { Widgets.hay(applicationContext) }
        // Con la app abierta, un aviso que llega pone al día la campanita y la pantalla (willPresent en iOS).
        EnPrimerPlano.alAviso = { lifecycleScope.launch { estado.version += 1; estado.refrescar() } }
        // Renovaciones y compras de Google Play que quedaron sin registrar.
        Compras.escuchar(applicationContext, estado)
        atender(intent)
        setContent {
            TemaTL(Preferencias.texto("tema") ?: "auto") {
                CompositionLocalProvider(LocalEstado provides estado) { Raiz() }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        atender(intent)
    }

    override fun onStart() {
        super.onStart()
        if (::estado.isInitialized && estado.fase == Estado.Fase.adentro) {
            estado.enVivo.arrancar(estado)
            lifecycleScope.launch { estado.refrescar(); estado.version += 1 }
        }
    }

    override fun onResume() {
        super.onResume()
        // Se cerró la Custom Tab sin terminar el ingreso: se suelta el botón, sin error (como cancelar en iOS).
        if (::estado.isInitialized && estado.ingresando && !vueltaClerk) estado.ingresando = false
    }

    override fun onStop() {
        super.onStop()
        if (::estado.isInitialized) estado.enVivo.parar()
    }

    /** La vuelta de Clerk (ar.com.tengolibre.app://callback), un link tengolibre:// o un App Link. */
    private fun atender(i: Intent?) {
        val u = i?.data ?: return
        if (u.scheme == Clerk.ESQUEMA) {
            vueltaClerk = true
            estado.vueltaClerk(u.getQueryParameter("rotating_token_nonce"), applicationContext)
            return
        }
        // Un aviso de otro negocio de la persona: se trabaja con ese.
        i.getStringExtra("negocio")?.let { id -> if (estado.negocio?.id != id) estado.negocios.firstOrNull { it.id == id }?.let { estado.usar(it) } }
        if (i.getStringExtra("tipo") == "resena") { estado.abrir(Ruta.resenas); return }
        Ruta.de(u.toString())?.let { estado.abrir(it) }
    }

    private suspend fun pedirPermisoAvisos(): Boolean {
        if (Build.VERSION.SDK_INT < 33) return true
        if (checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return true
        val d = CompletableDeferred<Boolean>()
        permiso = d
        pedirPermiso.launch(Manifest.permission.POST_NOTIFICATIONS)
        return d.await()
    }
}
