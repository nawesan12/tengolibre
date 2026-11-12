package ar.com.tengolibre.app

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.app.Registro
import ar.com.tengolibre.app.nucleo.Api
import ar.com.tengolibre.app.nucleo.ApiDePrueba
import ar.com.tengolibre.app.nucleo.ModoPrueba
import ar.com.tengolibre.app.nucleo.NegocioDeLaPersona
import ar.com.tengolibre.app.nucleo.Persona
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.nucleo.Turno
import ar.com.tengolibre.app.pantallas.agenda.Agenda
import ar.com.tengolibre.app.pantallas.avisos.Avisos
import ar.com.tengolibre.app.pantallas.avisos.Compartir
import ar.com.tengolibre.app.pantallas.buscar.Buscar
import ar.com.tengolibre.app.pantallas.cliente.Lugares
import ar.com.tengolibre.app.pantallas.cliente.MisTurnos
import ar.com.tengolibre.app.pantallas.cliente.PrincipalCliente
import ar.com.tengolibre.app.pantallas.cliente.Reservar
import ar.com.tengolibre.app.pantallas.clientes.Clientes
import ar.com.tengolibre.app.pantallas.clientes.FichaCliente
import ar.com.tengolibre.app.pantallas.espera.ListaEspera
import ar.com.tengolibre.app.pantallas.hoy.Hoy
import ar.com.tengolibre.app.pantallas.hoy.PlataHoy
import ar.com.tengolibre.app.pantallas.hoy.Recorrido
import ar.com.tengolibre.app.pantallas.ingreso.*
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.mas.HojaBorrarCuenta
import ar.com.tengolibre.app.pantallas.mas.Mas
import ar.com.tengolibre.app.pantallas.mas.Potencia
import ar.com.tengolibre.app.pantallas.principal.Destinos
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.pantallas.principal.Navegacion
import ar.com.tengolibre.app.pantallas.principal.NuevoTurnoInicio
import ar.com.tengolibre.app.pantallas.principal.Principal
import ar.com.tengolibre.app.pantallas.resenas.Resenas
import ar.com.tengolibre.app.pantallas.turno.Cobrar
import ar.com.tengolibre.app.pantallas.turno.DetalleTurno
import ar.com.tengolibre.app.pantallas.turno.NuevoTurno
import ar.com.tengolibre.app.pantallas.turno.Reprogramar
import ar.com.tengolibre.app.tema.C
import ar.com.tengolibre.app.tema.TemaTL
import com.github.takahirom.roborazzi.captureRoboImage
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** Clon de TengoLibreTests/PantallasTests.swift: monta cada pantalla contra la API simulada (pruebas/respuestas.json),
 *  a 393 × 852 dp, y guarda la captura en claro y oscuro (build/capturas/<Nombre>-claro|oscuro.png, los mismos nombres
 *  que iOS) para compararla con la de iOS (herramientas/comparar.sh).
 *  Falla si alguna pantalla se rompe al montarse. Al sumar una pantalla, sumarla acá (y en iOS). */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w393dp-h852dp-xxhdpi")
class PantallasTest {
    @get:Rule val compose = createComposeRule()

    private fun estado(rol: String = "duenio", negocios: Int = 1, extra: Map<String, String> = emptyMap()): Estado {
        val texto = { javaClass.getResource("/pruebas/respuestas.json")!!.readText() }
        ModoPrueba.respuestas = texto
        ModoPrueba.activo = true // Play Billing no se conecta en los tests
        Preferencias.borrarTodo()
        Preferencias.ponerBool("recorridoVisto", true)
        val api = Api("https://api.prueba", "prueba", OkHttpClient.Builder().addInterceptor(ApiDePrueba(extra, texto)).build())
        val e = Estado(api)
        e.persona = Persona("u1", "laura@lumi.test", "Laura Díaz", "")
        e.negocios = (1..negocios).map { NegocioDeLaPersona("n$it", "lumi$it".replace("lumi1", "lumi"), if (it == 1) "Lumi Estética" else "Lumi Palermo $it", rol) }
        e.usar(e.negocios.first())
        return e
    }

    /** Deja correr la carga: las respuestas llegan por el hilo de OkHttp y vuelven al principal, una tras otra
     *  (Hoy encadena una decena). El reloj de Compose avanza a la par. */
    private fun asentar(ms: Long) {
        repeat(12) {
            Thread.sleep(60)
            compose.mainClock.advanceTimeBy(ms / 12)
            compose.waitForIdle()
        }
    }

    private fun turno(e: Estado): Turno = runBlocking { e.api.get("/v1/panel/turnos/t1") }

    /** Monta la vista con el Estado y una Navegacion (NavigationStack + .destinos() en iOS) y captura claro y oscuro. */
    private fun montar(nombre: String, e: Estado = estado(), espera: Long = 1500, vista: @Composable () -> Unit) {
        Registro.instalar()
        val tema = mutableStateOf("claro")
        val nav = Navegacion()
        compose.setContent {
            TemaTL(tema.value) { CompositionLocalProvider(LocalEstado provides e, LocalNavegacion provides nav) { vista() } }
        }
        for (t in listOf("claro", "oscuro")) {
            tema.value = t
            asentar(espera)
            compose.onRoot().captureRoboImage("build/capturas/$nombre-$t.png")
        }
    }

    /** Lo que en iOS va en una hoja (ColumnScope acá): a pantalla completa, con el fondo de la hoja (Hoja lo pone). */
    @Composable private fun EnHoja(contenido: @Composable ColumnScope.() -> Unit) = Column(Modifier.fillMaxSize().background(C.hoja)) { contenido() }

    @Test fun decodificaTurno() = assertEquals("Sofía Méndez", turno(estado()).cliente.nombre)
    @Test fun hoy() = montar("Hoy") { Hoy() }
    @Test fun hoyProfesional() = montar("HoyProfesional", estado("profesional")) { Hoy() }
    @Test fun hoyVacio() = montar("HoyVacio", estado(extra = mapOf("/v1/panel/turnos" to """{"turnos": []}"""))) { Hoy() }
    @Test fun agenda() = montar("Agenda") { Agenda() }
    @Test fun espera() = montar("Espera") { ListaEspera() }
    @Test fun clientes() = montar("Clientes") { Clientes() }
    @Test fun ficha() = montar("Ficha") { FichaCliente("c1") }
    @Test fun resenas() = montar("Resenas") { Resenas() }
    @Test fun mas() = montar("Mas") { Mas() }
    @Test fun masProfesional() = montar("MasProfesional", estado("profesional")) { Mas() }
    @Test fun masRecepcion() = montar("MasRecepcion", estado("recepcion")) { Mas() }
    @Test fun detalle() { val e = estado(); val t = turno(e); montar("Detalle", e) { DetalleTurno(t) {} } }
    @Test fun nuevoTurno() = montar("NuevoTurno") { EnHoja { NuevoTurno(NuevoTurnoInicio()) {} } }
    @Test fun cobrar() { val e = estado(); val t = turno(e); montar("Cobrar", e) { EnHoja { Cobrar(t, null, {}, {}) } } }
    @Test fun reprogramar() { val e = estado(); val t = turno(e); montar("Reprogramar", e) { EnHoja { Reprogramar(t, {}, {}) } } }
    @Test fun avisos() = montar("Avisos") { EnHoja { Avisos {} } }
    @Test fun compartir() = montar("Compartir") { EnHoja { Compartir {} } }

    /** Las 27 secciones de «Más», como si se las hubiese tocado (NavigationLinkDirecto en iOS). Los nombres son los de
     *  iOS: String(describing:) del caso sin lo que no es letra o número. */
    @Test fun secciones() {
        val todas = listOf(
            "primerosPasos" to DestinoMas.primerosPasos, "resenas" to DestinoMas.resenas, "suscripcion" to DestinoMas.suscripcion,
            "diasLibres" to DestinoMas.diasLibres, "cobros" to DestinoMas.cobros, "servicios" to DestinoMas.servicios, "equipo" to DestinoMas.equipo,
            "profesionalp1" to DestinoMas.profesional("p1"), "miHorario" to DestinoMas.miHorario, "membresias" to DestinoMas.membresias,
            "cupones" to DestinoMas.cupones, "referidos" to DestinoMas.referidos, "penalidades" to DestinoMas.penalidades, "etiquetas" to DestinoMas.etiquetas,
            "caja" to DestinoMas.caja, "reportes" to DestinoMas.reportes, "miWeb" to DestinoMas.miWeb, "plantillas" to DestinoMas.plantillas,
            "dominio" to DestinoMas.dominio, "integraciones" to DestinoMas.integraciones, "configuracion" to DestinoMas.configuracion,
            "avisosClientes" to DestinoMas.avisosClientes, "whatsapp" to DestinoMas.whatsapp, "auditoria" to DestinoMas.auditoria,
            "sucursales" to DestinoMas.sucursales, "cuenta" to DestinoMas.cuenta, "recorrido" to DestinoMas.recorrido,
        )
        Registro.instalar()
        val e = estado()
        val tema = mutableStateOf("claro")
        val actual = mutableStateOf<Any>(DestinoMas.primerosPasos)
        compose.setContent {
            TemaTL(tema.value) { CompositionLocalProvider(LocalEstado provides e, LocalNavegacion provides Navegacion()) { Destinos.destino(actual.value) } }
        }
        for ((nombre, d) in todas) {
            actual.value = d
            for (t in listOf("claro", "oscuro")) {
                tema.value = t
                asentar(1000)
                compose.onRoot().captureRoboImage("build/capturas/Seccion-$nombre-$t.png")
            }
        }
    }

    @Test fun recorrido() = montar("Recorrido", espera = 1000) { Recorrido {} }
    @Test fun recorridoProfesional() = montar("RecorridoProfesional", estado("profesional"), 1000) { Recorrido {} }
    @Test fun buscar() = montar("Buscar", espera = 1000) { EnHoja { Buscar {} } }
    @Test fun potencia() = montar("Potencia") { Potencia() }
    @Test fun plataHoy() = montar("PlataHoy") { Column(Modifier.verticalScroll(rememberScrollState())) { PlataHoy() } }
    @Test fun fichaFiel() = montar("FichaFiel", estado(extra = mapOf("/v1/panel/membresias/planes" to """{"planes": []}"""))) { FichaCliente("c1") }
    @Test fun arranque() = montar("Arranque") { PantallaArranque() }
    @Test fun ingreso() = montar("Ingreso", espera = 1000) { Ingreso() }
    @Test fun ingresoMail() = montar("IngresoMail", espera = 1000) { EnHoja { IngresoMail {} } }
    @Test fun recuperarAcceso() = montar("RecuperarAcceso", espera = 1000) { EnHoja { RecuperarAcceso("laura@lumi.test") {} } }
    @Test fun borrarCuenta() = montar("BorrarCuenta", espera = 1000) { EnHoja { HojaBorrarCuenta {} } }
    @Test fun elegirModo() = montar("ElegirModo", espera = 1000) { ElegirModo() }
    @Test fun elegirNegocio() = montar("ElegirNegocio", estado(negocios = 3), 1000) { ElegirNegocio() }
    @Test fun permisos() = montar("Permisos", espera = 1000) { PermisoAvisos() }
    @Test fun alta() = montar("Alta", espera = 1000) { Alta() }
    @Test fun principal() = montar("Principal") { Principal() }
    @Test fun principalProfesional() = montar("PrincipalProfesional", estado("profesional")) { Principal() }
    @Test fun cliente() { val e = estado(); MisTurnos.codigos = emptyList(); montar("Cliente", e) { PrincipalCliente() } }
    @Test fun reservar() = montar("Reservar") { EnHoja { Reservar("lumi", {}, {}) } }
    @Test fun lugares() = montar("Lugares") { Lugares {} }
}
