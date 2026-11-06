package ar.com.tengolibre.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import ar.com.tengolibre.app.nucleo.Guardado
import ar.com.tengolibre.app.nucleo.Llavero
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.pantallas.Muestra
import ar.com.tengolibre.app.tema.Haptico
import ar.com.tengolibre.app.tema.TemaTL
import java.lang.ref.WeakReference

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Preferencias.contexto = applicationContext
        Guardado.contexto = applicationContext
        Llavero.contexto = applicationContext
        enableEdgeToEdge()
        Haptico.vista = WeakReference(window.decorView)
        setContent { TemaTL { Muestra() } }
    }
}
