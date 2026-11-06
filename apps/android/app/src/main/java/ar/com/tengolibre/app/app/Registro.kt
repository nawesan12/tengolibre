package ar.com.tengolibre.app.app

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.ui.Modifier
import ar.com.tengolibre.app.pantallas.agenda.Agenda
import ar.com.tengolibre.app.pantallas.avisos.Avisos
import ar.com.tengolibre.app.pantallas.avisos.Compartir
import ar.com.tengolibre.app.pantallas.buscar.Buscar
import ar.com.tengolibre.app.pantallas.clientes.Clientes
import ar.com.tengolibre.app.pantallas.clientes.DestinoClientes
import ar.com.tengolibre.app.pantallas.clientes.FichaCliente
import ar.com.tengolibre.app.pantallas.espera.ListaEspera
import ar.com.tengolibre.app.pantallas.hoy.Hoy
import ar.com.tengolibre.app.pantallas.hoy.Recorrido
import ar.com.tengolibre.app.pantallas.hoy.RecorridoOtraVez
import ar.com.tengolibre.app.pantallas.mas.*
import ar.com.tengolibre.app.pantallas.principal.Destinos
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.pantallas.principal.Pestana
import ar.com.tengolibre.app.pantallas.resenas.Resenas
import ar.com.tengolibre.app.pantallas.turno.DetalleTurno
import ar.com.tengolibre.app.pantallas.turno.NuevoTurno

/** Une la navegación de Principal con las pantallas (en iOS, los navigationDestination de Mas.swift y las hojas de
 *  Principal.swift). Va aparte para que Principal no dependa de cada sección. */
object Registro {
    private var listo = false

    fun instalar() {
        if (listo) return
        listo = true
        Destinos.raiz = { p ->
            when (p) {
                Pestana.hoy -> Hoy()
                Pestana.agenda -> Agenda()
                Pestana.espera -> ListaEspera()
                Pestana.clientes -> Clientes()
                Pestana.resenas -> Resenas()
                Pestana.mas -> Mas()
            }
        }
        Destinos.destino = { d ->
            when (d) {
                is DestinoClientes.ficha -> FichaCliente(d.id)
                is DestinoMas -> DestinoDeMas(d)
                else -> {}
            }
        }
        Destinos.detalleTurno = { t -> val nav = LocalNavegacion.current; DetalleTurno(t) { nav.turno = null } }
        Destinos.nuevoTurno = { i -> val nav = LocalNavegacion.current; Column(Modifier.fillMaxSize()) { NuevoTurno(i) { nav.nuevo = null } } }
        Destinos.compartir = { val nav = LocalNavegacion.current; Column(Modifier.fillMaxSize()) { Compartir { nav.compartir = false } } }
        Destinos.buscar = { val nav = LocalNavegacion.current; Column(Modifier.fillMaxSize()) { Buscar { nav.buscar = false } } }
        Destinos.avisos = { val nav = LocalNavegacion.current; Column(Modifier.fillMaxSize()) { Avisos { nav.avisos = false } } }
        Destinos.recorrido = { alTerminar -> Recorrido(alTerminar) }
        Destinos.destinoSuscripcion = DestinoMas.suscripcion
        Destinos.destinoCaja = DestinoMas.caja
        Destinos.destinoResenas = DestinoMas.resenas
    }
}

@androidx.compose.runtime.Composable
private fun DestinoDeMas(d: DestinoMas) {
    when (d) {
        DestinoMas.primerosPasos -> Potencia()
        DestinoMas.resenas -> Resenas()
        DestinoMas.suscripcion -> SuscripcionVista()
        DestinoMas.diasLibres -> DiasLibres()
        DestinoMas.cobros -> Cobros()
        DestinoMas.servicios -> Servicios()
        DestinoMas.equipo -> Equipo()
        is DestinoMas.profesional -> FichaProfesional(d.id)
        DestinoMas.miHorario -> MiHorario()
        DestinoMas.membresias -> Membresias()
        DestinoMas.cupones -> Cupones()
        DestinoMas.referidos -> Referidos()
        DestinoMas.penalidades -> Penalidades()
        DestinoMas.etiquetas -> Etiquetas()
        DestinoMas.caja -> CajaDelDia()
        DestinoMas.reportes -> Reportes()
        DestinoMas.miWeb -> MiWeb()
        DestinoMas.plantillas -> Plantillas()
        DestinoMas.dominio -> DominioVista()
        DestinoMas.integraciones -> Integraciones()
        DestinoMas.configuracion -> Configuracion()
        DestinoMas.avisosClientes -> AvisosClientes()
        DestinoMas.whatsapp -> WhatsAppVista()
        DestinoMas.auditoria -> Auditoria()
        DestinoMas.sucursales -> Sucursales()
        DestinoMas.cuenta -> Cuenta()
        DestinoMas.recorrido -> RecorridoOtraVez()
    }
}
