package ar.com.tengolibre.app.pantallas.espera

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.Oferta
import ar.com.tengolibre.app.pantallas.hoy.Troquel
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de Pantallas/Espera/ListaEspera.swift.

/** 12 Lista de espera: Esperando · Avisados · Vencidos. Cada persona con su rango de días y momento, el servicio y con
 *  quién, desde cuándo espera y, si hay un hueco que le sirve, «Avisar». Arriba, los huecos que se están ofreciendo
 *  de a una persona por vez (Frenar · Cargarlo yo). */
@Composable
fun ListaEspera() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var tab by remember { mutableStateOf("esperando") }
    var espera by remember { mutableStateOf<List<Anotado>>(emptyList()) }
    var ofertas by remember { mutableStateOf<List<Oferta>>(emptyList()) }
    var sugeridos by remember { mutableStateOf<Map<String, List<HorarioLibre>>>(emptyMap()) }
    var profesionales by remember { mutableStateOf<List<Profesional>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var borrar by remember { mutableStateOf<Anotado?>(null) }
    val hoy = Formato.diaISO(Instant.now())
    fun tabDe(a: Anotado) = if (a.avisadoEn != null) "avisados" else if (a.hasta < hoy) "vencidos" else "esperando"
    val lista = espera.filter { tabDe(it) == tab }
    val esperando = espera.count { tabDe(it) == "esperando" }
    val conHueco = espera.count { tabDe(it) == "esperando" && !sugeridos[it.id].isNullOrEmpty() }

    @Serializable data class Espera(val espera: List<Anotado> = emptyList())
    @Serializable data class Ofertas(val ofertas: List<Oferta> = emptyList())
    @Serializable data class Libres(val horarios: List<HorarioLibre> = emptyList())
    @Serializable data class Pros(val profesionales: List<Profesional> = emptyList())

    suspend fun libresPara(a: Anotado): List<HorarioLibre> {
        var d = runCatching { java.time.LocalDate.parse(maxOf(a.desde, hoy)) }.getOrNull() ?: return emptyList()
        val fin = runCatching { java.time.LocalDate.parse(a.hasta) }.getOrNull() ?: return emptyList()
        repeat(7) {
            if (d.isAfter(fin)) return emptyList()
            val q = buildList { add("servicio" to a.servicioId); add("fecha" to d.toString()); a.profesionalId?.let { add("profesional" to it) } }
            val r = estado.api.opcional<Libres>("/v1/panel/libres", q)
            if (r != null && r.horarios.isNotEmpty()) {
                val m = r.horarios.filter { h -> val min = Formato.minutosDelDia(h.inicio); if (a.momento == "manana") min < 13 * 60 else if (a.momento == "tarde") min >= 13 * 60 else true }
                if (m.isNotEmpty()) return m.take(6)
            }
            d = d.plusDays(1)
        }
        return emptyList()
    }

    suspend fun cargar() {
        estado.api.opcional<Espera>("/v1/panel/espera", listOf("todos" to "1"))?.let { espera = it.espera }
        ofertas = (estado.api.opcional<Ofertas>("/v1/panel/espera/ofertas")?.ofertas ?: emptyList()).filter { it.estado == "ofreciendo" }
        estado.api.opcional<Pros>("/v1/panel/profesionales")?.let { profesionales = it.profesionales }
        cargando = false
        // Los primeros horarios libres dentro de lo que pidió cada una (como la web: 7 días, hasta 6).
        for (a in espera.filter { tabDe(it) == "esperando" }.take(30)) if (!sugeridos.containsKey(a.id)) sugeridos = sugeridos + (a.id to libresPara(a))
    }
    LaunchedEffect(estado.version) { cargar() }

    @Serializable data class Aviso(val inicio: String?, val nota: String)
    fun avisar(a: Anotado, h: HorarioLibre) = alcance.launch {
        try {
            estado.api.hacer("POST", "/v1/panel/espera/${a.id}/avisar", Aviso(Fecha.texto(h.inicio), ""))
            Haptico.exito()
            estado.avisar("Le avisamos a ${a.nombre.split(" ").firstOrNull().orEmpty()}", "${Formato.dia(h.inicio)} a las ${h.hora}")
            cargar()
        } catch (e: Exception) { estado.avisar(e) }
    }
    fun sacar(a: Anotado) = alcance.launch {
        try { estado.api.hacer("DELETE", "/v1/panel/espera/${a.id}"); espera = espera.filter { it.id != a.id } } catch (e: Exception) { estado.avisar(e) }
    }
    @Serializable data class Cargada(val oferta: Oferta)
    fun ofertaAccion(o: Oferta, que: String) = alcance.launch {
        try {
            if (que == "cargar") {
                val r: Cargada = estado.api.enviar("POST", "/v1/panel/espera/ofertas/${o.id}/cargar", emptyMap<String, String>())
                nav.nuevo = NuevoTurnoInicio(profesionalId = r.oferta.profesionalId, servicioId = r.oferta.servicioId, inicio = r.oferta.inicio)
            } else {
                estado.api.pedir("POST", "/v1/panel/espera/ofertas/${o.id}/frenar")
                estado.avisar("Dejamos de ofrecerlo", "Quien tenía el link vuelve a esperar.")
            }
            cargar()
        } catch (e: Exception) { estado.avisar(e) }
    }

    Refrescable({ cargar() }, Modifier.fillMaxSize().background(C.fondo)) { Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Lista de espera", "$esperando esperando${if (conHueco > 0) " · a $conHueco le${if (conHueco == 1) "" else "s"} sirve un hueco" else ""}") {
            Segmentado(listOf("esperando" to "Esperando", "avisados" to "Avisados", "vencidos" to "Vencidos"), tab, { tab = it })
        }
        Cuerpo {
            if (tab == "esperando") ofertas.forEach { o ->
                Hueco(Modifier.fillMaxWidth(), 20.dp, alinear = Alignment.TopStart) {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                            Icono("megaphone", 17f, C.mentaTexto)
                            Texto("Ofreciendo ${Formato.horaDe(o.inicio)} · ${o.profesional}", Tipo.lato(16f, Peso.Bold), C.huecoTinta)
                        }
                        o.actual?.let { a ->
                            Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                                Texto("A ${a.nombre}", Tipo.lato(15f), C.huecoTinta)
                                a.venceEn?.let { Texto("· vence a las ${Formato.horaDe(it)}", Tipo.lato(13f), C.sub) }
                            }
                        }
                        TextoRotulo(o.sigue?.let { "Sigue: ${it.nombre}" } ?: "Es la última")
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Box(Modifier.weight(1f)) { BotonContorno("Frenar", { ofertaAccion(o, "frenar") }, alto = 44.dp, tam = 15f) }
                            Box(Modifier.weight(1f)) { BotonContorno("Cargarlo yo", { ofertaAccion(o, "cargar") }, alto = 44.dp, tam = 15f) }
                        }
                    }
                }
            }
            if (cargando && espera.isEmpty()) repeat(3) { Esqueleto(120.dp, 20.dp) }
            else if (lista.isEmpty()) VacioQueEnsena(
                if (tab == "esperando") "Nadie esperando" else "Nada por acá",
                if (tab == "esperando") "Nadie está esperando un lugar" else if (tab == "avisados") "Todavía no avisaste a nadie" else "No hay pedidos vencidos",
                "Cuando un día está lleno, en tu página aparece «Avisame si se libera». Si alguien cancela, se lo ofrecemos solos.",
            )
            else lista.forEachIndexed { i, a ->
                val match = sugeridos[a.id]?.firstOrNull()
                val quien = match?.profesionales?.firstOrNull()?.let { id -> profesionales.firstOrNull { it.id == id }?.nombre } ?: ""
                Column(Modifier.entrada(i).fillMaxWidth().tarjeta().padding(14.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(Modifier.height(IntrinsicSize.Min), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Row(Modifier.fillMaxHeight()) {
                            Column(Modifier.width(64.dp).padding(end = 12.dp), verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically)) {
                                Texto(rango(a), Tipo.lato(15f, Peso.Black, numeros = true), C.tinta, lineas = 2)
                                Texto(a.momentoTexto, Tipo.lato(12f), C.sub)
                            }
                            Troquel()
                        }
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Texto(a.nombre, Tipo.lato(17f, Peso.Bold), C.tinta)
                            Texto("${a.servicio} · con ${a.profesional ?: "cualquiera"}", Tipo.lato(13f), C.sub)
                            Texto(if (a.avisadoEn != null) textoAvisado(a) else "Espera desde el ${Formato.dia(a.creadoEn)}", Tipo.lato(12f), C.sub)
                        }
                        val tel = a.telefono.filter { it in "+0123456789" }
                        MenuBoton(buildList {
                            if (tel.isNotEmpty()) add(OpcionDeMenu("Llamar", "phone") { estado.abrirLink("tel:$tel") })
                            add(OpcionDeMenu("Cargarle un turno", "plus") {
                                nav.nuevo = NuevoTurnoInicio(profesionalId = a.profesionalId, servicioId = a.servicioId, cliente = ClienteDelTurno(null, a.nombre, a.telefono, a.email), esperaId = a.id)
                            })
                            add(OpcionDeMenu("Sacar de la lista", "trash", destructiva = true) { borrar = a })
                        }, "Más acciones para ${a.nombre}") { Box(Modifier.size(44.dp), contentAlignment = Alignment.Center) { Icono("ellipsis", 17f, C.sub) } }
                    }
                    if (match != null && tab == "esperando") {
                        Hueco(Modifier.fillMaxWidth(), 14.dp) {
                            Row(Modifier.fillMaxWidth().padding(start = 12.dp, top = 8.dp, bottom = 8.dp, end = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                                Box(Modifier.weight(1f).offset(x = (-4).dp)) {
                                    PastillaPapel(8.dp) {
                                        Texto("${if (Formato.diaISO(match.inicio) == hoy) "HOY" else Formato.dia(match.inicio).uppercase()} ${match.hora}${if (quien.isEmpty()) "" else " · ${quien.uppercase()}"}",
                                            Tipo.lato(13f, Peso.Bold, numeros = true), C.huecoTinta, tracking = 0.8.sp)
                                    }
                                }
                                Box(Modifier.atenuado(estado.soloLectura)) { BotonMentaChico("Avisar", { if (!estado.soloLectura) avisar(a, match) }) }
                            }
                        }
                    }
                }
            }
        }
    } }
    val b = borrar
    Confirmacion(b != null, "¿Sacar a ${b?.nombre.orEmpty()} de la lista?", "", listOf(OpcionConfirmacion("Sacar de la lista", destructiva = true) { b?.let { sacar(it) } }), { borrar = null }, volver = "Cancelar")
}

private fun rango(a: Anotado): String {
    val d = Fecha.iso(a.desde) ?: return ""
    val h = Fecha.iso(a.hasta) ?: return ""
    if (a.desde == a.hasta) return Formato.dia(d)
    return "${Formato.enArgentina(d).dayOfMonth} al ${Formato.dia(h)}"
}

private fun textoAvisado(a: Anotado): String {
    val r = mapOf("tomada" to "lo tomó", "paso" to "dijo que no le sirve", "vencida" to "no respondió", "cancelada" to "se cerró la oferta", "ofrecida" to "tiene el link")[a.resultado ?: ""]
    return "Avisada el ${Formato.dia(a.avisadoEn!!)}${r?.let { " · $it" } ?: ""}"
}
