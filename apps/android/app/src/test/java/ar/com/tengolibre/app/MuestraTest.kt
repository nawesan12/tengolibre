package ar.com.tengolibre.app

import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import ar.com.tengolibre.app.pantallas.Muestra
import ar.com.tengolibre.app.tema.TemaTL
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

// Las piezas del tema en claro y oscuro, a 393 × 852 dp (el iPhone de referencia de iOS).
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w393dp-h852dp-xxhdpi")
class MuestraTest {
    @get:Rule val compose = createComposeRule()

    @Test fun claro() {
        compose.setContent { TemaTL("claro") { Muestra() } }
        compose.onRoot().captureRoboImage("build/capturas/muestra-claro.png")
    }

    @Test fun oscuro() {
        compose.setContent { TemaTL("oscuro") { Muestra() } }
        compose.onRoot().captureRoboImage("build/capturas/muestra-oscuro.png")
    }
}
