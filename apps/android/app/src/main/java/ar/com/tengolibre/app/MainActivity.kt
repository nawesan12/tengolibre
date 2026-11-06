package ar.com.tengolibre.app

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
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
import ar.com.tengolibre.app.nucleo.Guardado
import ar.com.tengolibre.app.nucleo.Llavero
import ar.com.tengolibre.app.nucleo.ModoPrueba
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.tema.Haptico
import ar.com.tengolibre.app.tema.TemaTL
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.launch
import java.lang.ref.WeakReference

/** La única actividad (TengoLibreApp en iOS): arma el Estado, recibe los links, la vuelta de Clerk y el permiso de
 *  avisos, y prende o apaga el en vivo al entrar y salir. */
class MainActivity : ComponentActivity() {
    private lateinit var estado: Estado
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
        enableEdgeToEdge()
        Haptico.vista = WeakReference(window.decorView)
        estado = Estado()
        estado.abrirPestana = { url -> CustomTabsIntent.Builder().setShowTitle(true).build().launchUrl(this, Uri.parse(url)) }
        estado.alPedirPermisoPush = { pedirPermisoAvisos() }
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

    override fun onStop() {
        super.onStop()
        if (::estado.isInitialized) estado.enVivo.parar()
    }

    /** La vuelta de Clerk (ar.com.tengolibre.app://callback), un link tengolibre:// o un App Link. */
    private fun atender(i: Intent?) {
        val u = i?.data ?: return
        if (u.scheme == Clerk.ESQUEMA) {
            estado.vueltaClerk(u.getQueryParameter("rotating_token_nonce"), applicationContext)
            return
        }
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
