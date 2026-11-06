package ar.com.tengolibre.app

import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.Ruta
import ar.com.tengolibre.app.nucleo.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Before
import org.junit.Test
import java.time.Instant

// Los mismos casos que RutasTests (iOS) más el núcleo: fechas de Go, formato, huecos del día y la API simulada.
class NucleoTest {
    private val principal = StandardTestDispatcher()
    @Before fun antes() { Dispatchers.setMain(principal) }
    @After fun despues() { Dispatchers.resetMain() }

    @Test fun rutas() {
        assertEquals(Ruta.turno("t1"), Ruta.deTexto("turnos/t1"))
        assertEquals(Ruta.hoy, Ruta.deTexto("turnos"))
        assertEquals(Ruta.espera, Ruta.deTexto("lista-espera"))
        assertEquals(Ruta.avisos, Ruta.deTexto("notificaciones"))
        assertEquals(Ruta.turnoCliente("ABC"), Ruta.deTexto("turno-cliente/ABC"))
        assertEquals(null, Ruta.deTexto("cualquiera"))
    }

    @Test fun dia() {
        assertEquals(570, Dia.aMin("9:30"))
        assertEquals("9:05", Dia.hhmm(545))
        val libres = Dia.restar(listOf(Dia.Tramo(540, 720)), listOf(Dia.Tramo(600, 630)), desde = 0)
        assertEquals(listOf(Dia.Tramo(540, 600), Dia.Tramo(630, 720)), libres)
    }

    @Test fun fechasDeGo() {
        assertNotNull(Fecha.iso("2026-11-04T14:32:10.123456789-03:00"))
        assertNotNull(Fecha.iso("2026-11-04T17:32:10Z"))
        assertEquals("2026-11-04", Formato.diaISO(Fecha.iso("2026-11-04")!!))
    }

    @Test fun formato() {
        assertEquals("$ 24.020", Formato.pesos(24020))
        assertEquals("−$ 2.400", Formato.pesos(-2400))
        assertEquals("12,4 %", Formato.porcentaje(12.4))
        assertEquals("1 h 30", Formato.duracion(90))
        assertEquals("1 h 30 libre", Formato.libres(90))
    }

    @Test fun sse() {
        val e = EnVivo.parsear("event: turnos\ndata: {\"negocio\":\"n1\",\"tipo\":\"turnos\",\"id\":\"t1\",\"op\":\"actualizado\"}\n\n")
        assertEquals("turnos", e.single().second?.tipo)
    }

    @Test fun arrancaConLaApiSimulada() = runTest(principal) {
        ModoPrueba.respuestas = { javaClass.getResource("/pruebas/respuestas.json")!!.readText() }
        Preferencias.borrarTodo()
        Preferencias.ponerTexto("modo", "negocio")
        Preferencias.ponerBool("permisosVistos", true)
        val estado = Estado(ModoPrueba.api(), this)
        estado.arrancar()
        assertEquals(Estado.Fase.adentro, estado.fase)
        assertEquals("Lumi Estética", estado.negocio?.nombre)
        assertEquals(2, estado.sinLeer)
        val turnos: Turnos = estado.api.get("/v1/panel/turnos")
        assertEquals(4, turnos.turnos.size)
        assert(turnos.turnos[1].inicio.isBefore(Instant.now()))
    }

    @kotlinx.serialization.Serializable data class Turnos(val turnos: List<Turno>)
}
