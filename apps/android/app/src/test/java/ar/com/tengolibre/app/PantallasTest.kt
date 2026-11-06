package ar.com.tengolibre.app

import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.ModoPrueba
import ar.com.tengolibre.app.nucleo.NegocioDeLaPersona
import ar.com.tengolibre.app.nucleo.Persona
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.pantallas.ingreso.*
import ar.com.tengolibre.app.pantallas.principal.Principal
import ar.com.tengolibre.app.tema.TemaTL
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** Clon de TengoLibreTests/PantallasTests.swift: monta cada pantalla contra la API simulada (pruebas/respuestas.json),
 *  a 393 × 852 dp, y guarda la captura en claro y oscuro (build/capturas/<Nombre>-claro|oscuro.png, los mismos nombres que iOS)
 *  para compararla con la de iOS (herramientas/comparar.sh).
 *  Falla si alguna pantalla se rompe al montarse. Al sumar una pantalla, sumarla acá (y en iOS). */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w393dp-h852dp-xxhdpi")
class PantallasTest {
    @get:Rule val compose = createComposeRule()

    private fun estado(rol: String = "duenio", negocios: Int = 1): Estado {
        ModoPrueba.respuestas = { javaClass.getResource("/pruebas/respuestas.json")!!.readText() }
        Preferencias.borrarTodo()
        val e = Estado(ModoPrueba.api())
        e.persona = Persona("u1", "laura@lumi.test", "Laura Díaz", "")
        e.negocios = (1..negocios).map { NegocioDeLaPersona("n$it", "lumi$it", if (it == 1) "Lumi Estética" else "Lumi Palermo $it", rol) }
        e.usar(e.negocios.first())
        return e
    }

    private fun montar(nombre: String, e: Estado = estado(), vista: @Composable () -> Unit) {
        val tema = mutableStateOf("claro")
        compose.setContent { TemaTL(tema.value) { CompositionLocalProvider(LocalEstado provides e) { vista() } } }
        for (t in listOf("claro", "oscuro")) {
            tema.value = t
            compose.mainClock.advanceTimeBy(1500)
            compose.waitForIdle()
            compose.onRoot().captureRoboImage("build/capturas/$nombre-$t.png")
        }
    }

    @Test fun arranque() = montar("Arranque") { PantallaArranque() }
    @Test fun ingreso() = montar("Ingreso") { Ingreso() }
    @Test fun elegirModo() = montar("ElegirModo") { ElegirModo() }
    @Test fun elegirNegocio() = montar("ElegirNegocio", estado(negocios = 3)) { ElegirNegocio() }
    @Test fun permisos() = montar("Permisos") { PermisoAvisos() }
    @Test fun alta() = montar("Alta") { Alta() }
    @Test fun principal() { Preferencias.ponerBool("recorridoVisto", true); montar("Principal") { Principal() } }
    @Test fun principalProfesional() = montar("PrincipalProfesional", estado("profesional")) { Principal() }
}
