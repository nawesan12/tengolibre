package ar.com.tengolibre.app.pantallas.hoy

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.pantallas.mas.Bloque
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.mas.PantallaMas
import ar.com.tengolibre.app.pantallas.mas.Renglon
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.pantallas.principal.Pestana
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch

// Clon de Pantallas/Hoy/Guia.swift. La app guía sola (Hoy nunca queda sin caminos): atajos a todo lo del negocio,
// «Descubrí» con lo que todavía no se usa y el recorrido de la primera vez.

private data class Atajo(val simbolo: String, val titulo: String, val destino: DestinoMas?, val accion: (() -> Unit)? = null)

/** Los atajos de Hoy: lo que vive en «Más», a un toque y sin salir de Hoy. */
@Composable
fun AtajosHoy() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val g = estado.gestiona; val mostrador = g || estado.rol == "recepcion"
    val atajos = if (estado.esProfesional) listOf(
        Atajo("clock", "Mi horario", DestinoMas.miHorario),
        Atajo("beach.umbrella", "Días libres", DestinoMas.miHorario),
        Atajo("star.bubble", "Reseñas", null) { nav.pestana = Pestana.resenas },
        Atajo("calendar.badge.plus", "En mi calendario", DestinoMas.integraciones),
    ) else buildList {
        add(Atajo("banknote", "Caja de hoy", DestinoMas.caja))
        if (g) add(Atajo("creditcard", "Señas", DestinoMas.cobros))
        if (mostrador) add(Atajo("ticket", "Membresías", DestinoMas.membresias))
        if (g) add(Atajo("chart.bar", "Reportes", DestinoMas.reportes))
        if (mostrador) add(Atajo("tag", "Cupones", DestinoMas.cupones))
        if (g) {
            add(Atajo("sparkles", "IA", DestinoMas.integraciones))
            add(Atajo("globe", "Mi web", DestinoMas.miWeb))
            add(Atajo("person.2.badge.gearshape", "Equipo", DestinoMas.equipo))
        }
        add(Atajo("square.and.arrow.up", "Mi link", null) { nav.compartir = true })
    }
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            TextoRotulo("Tu negocio", modifier = Modifier.weight(1f))
            Texto("Ver todo", Tipo.lato(13f, Peso.Bold), C.mentaTexto, Modifier.clickable { nav.pestana = Pestana.mas })
        }
        Row(Modifier.offset(x = (-16).dp).fillMaxWidth().horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp, vertical = 2.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            atajos.forEachIndexed { i, a ->
                Column(
                    Modifier.entrada(i).size(104.dp, 96.dp).tarjeta(18.dp).clickable { Haptico.suave(); a.destino?.let { nav.pilaHoy.add(it) } ?: a.accion?.invoke() }.padding(12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Box(Modifier.size(36.dp).clip(RoundedCornerShape(10.dp)).background(C.mentaSuave), contentAlignment = Alignment.Center) { Icono(a.simbolo, 20f, C.mentaTexto) }
                    Texto(a.titulo, Tipo.lato(14f, Peso.Bold), C.tinta, lineas = 2)
                }
            }
            Spacer(Modifier.width(16.dp))
        }
    }
}

data class Idea(val id: String, val simbolo: String, val titulo: String, val texto: String, val accion: String, val destino: DestinoMas?)

/** «Descubrí»: una tarjeta por vez con algo que el negocio todavía no aprovecha, con su botón directo. */
@Composable
fun DescubrirHoy() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    var descartadas by remember { mutableStateOf(Preferencias.texto("descubrirDescartadas").orEmpty()) }
    val v = estado.vivo
    val ideas = buildList {
        if (estado.esProfesional) {
            add(Idea("calendario", "calendar.badge.plus", "Tus turnos en tu calendario", "Suscribite desde Google o Apple y te aparecen solos.", "Conectar", DestinoMas.integraciones))
            if (!v.widget) add(Idea("widget", "apps.iphone", "Tu próximo turno, sin abrir la app", "Sumá el widget de Tengo Libre a tu pantalla de inicio.", "Cómo se hace", DestinoMas.recorrido))
            return@buildList
        }
        if (!v.cargado) return@buildList
        if (estado.gestiona) {
            v.mp?.let { m -> if (m.disponible && !m.conectada) add(Idea("mp", "creditcard", "Cobrá señas al reservar", "Con Mercado Pago nadie reserva y no viene. La plata va directo a tu cuenta.", "Conectar", DestinoMas.cobros)) }
            if (v.iaActiva == false) add(Idea("ia", "sparkles", "Que te reserven desde ChatGPT y Claude", "Tus horarios libres quedan a mano de los asistentes de IA.", "Activar", DestinoMas.integraciones))
            if (v.clavesIA == 0) add(Idea("mcp", "bubble.left.and.text.bubble.right", "Manejá tu agenda hablando", "Conectá Claude o ChatGPT y pediles «¿qué tengo mañana?» o «mové a Sofía al jueves».", "Ver cómo", DestinoMas.integraciones))
            if (v.fotos == 0 || !v.logo) add(Idea("fotos", "photo.on.rectangle", "Sumá logo y fotos a tu página", "Las páginas con fotos reciben más reservas.", "Subir fotos", DestinoMas.miWeb))
            if (v.whatsapp == false) add(Idea("wa", "message", "Que confirmen solas por WhatsApp", "El día antes les pedimos que confirmen; si no contestan, te avisamos.", "Prender", DestinoMas.whatsapp))
        }
        if (estado.gestiona || estado.rol == "recepcion") {
            if (v.planes == 0) add(Idea("membresias", "ticket", "Vendé membresías", "Un abono mensual con visitas incluidas: plata fija todos los meses.", "Crear un plan", DestinoMas.membresias))
            if (v.recordatorios == 0) add(Idea("recordatorios", "envelope.badge", "Que vuelvan solas", "Un mail en su cumpleaños y otro si hace mucho que no vienen.", "Prender", DestinoMas.avisosClientes))
            if (v.cupones == 0) add(Idea("cupones", "tag", "Llená las horas flojas", "Mirá qué horarios quedan siempre vacíos y armá un cupón para esos.", "Ver mis horas muertas", DestinoMas.reportes))
        }
        if (!v.widget) add(Idea("widget", "apps.iphone", "Tu día en la pantalla de inicio", "El widget te muestra el próximo turno sin abrir la app.", "Cómo se hace", DestinoMas.recorrido))
        add(Idea("compartir", "qrcode", "Que te reserven solos", "Poné tu link en Instagram y WhatsApp, o imprimí el QR para el mostrador.", "Compartir mi link", null))
    }
    val fuera = descartadas.split(",").filter { it.isNotEmpty() }
    val visibles = ideas.filter { it.id !in fuera }
    AnimatedContent(visibles.firstOrNull(), transitionSpec = { (slideInHorizontally { it } + fadeIn()) togetherWith (slideOutHorizontally { -it } + fadeOut()) }, label = "descubri") { i ->
        if (i == null) return@AnimatedContent
        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(22.dp)).fondoPetroleo(C.bloquePetroleo).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Box(Modifier.size(44.dp).clip(RoundedCornerShape(12.dp)).background(Color.White.copy(alpha = 0.1f)), contentAlignment = Alignment.Center) { Icono(i.simbolo, 22f, C.menta) }
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Texto("DESCUBRÍ · ${ideas.size - visibles.size + 1} DE ${ideas.size}", Tipo.lato(11f, Peso.Black), C.menta, tracking = 0.8.sp)
                    Texto(i.titulo, Tipo.lato(17f, Peso.Bold), Color.White)
                    Texto(i.texto, Tipo.lato(14f), Color.White.copy(alpha = 0.8f))
                }
            }
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                BotonMenta(i.accion, {
                    Haptico.suave()
                    if (i.destino != null) nav.pilaHoy.add(i.destino) else if (i.id == "compartir") nav.compartir = true
                }, Modifier.weight(1f), alto = 44.dp, tam = 15f, conSombra = false)
                Texto("Ahora no", Tipo.lato(15f, Peso.Bold), Color.White.copy(alpha = 0.85f), Modifier.clickable {
                    descartadas = (fuera + i.id).joinToString(","); Preferencias.ponerTexto("descubrirDescartadas", descartadas)
                }.padding(horizontal = 12.dp, vertical = 12.dp))
            }
        }
    }
}

private data class Paso(val simbolo: String, val titulo: String, val texto: String, val puntos: List<String>)

/** El recorrido de la primera vez (y desde Más › «Qué podés hacer»): pocas pantallas, poco texto. Pantalla
 *  completa encima de todo (fullScreenCover en iOS). */
@Composable
fun Recorrido(alTerminar: () -> Unit) {
    val estado = LocalEstado.current
    // «La Isla Dinámica» de iOS acá es la notificación en curso.
    val pasos = if (estado.esProfesional) listOf(
        Paso("sun.max", "Tu día, de un vistazo", "Tus turnos de hoy en orden, con el que está pasando arriba.", listOf("Deslizá un turno para marcarlo atendido", "Tocalo para ver la clienta y llamarla")),
        Paso("beach.umbrella", "Tu horario y tus días libres", "Desde Más los pedís y te avisamos cuando los aprueban.", listOf("Bloqueá un rato o un día", "Pedí vacaciones sin mandar mensajes")),
        Paso("bell.badge", "Enterate al toque", "Cada reserva o cancelación te llega como aviso.", listOf("Sumá el widget a tu pantalla de inicio", "La notificación te muestra el turno en curso")),
    ) else listOf(
        Paso("sun.max", "Hoy, siempre al día", "Los turnos de hoy, los huecos libres y lo que hay que atender. Se actualiza solo.", listOf("Deslizá un turno: atendido o ausente", "Tocá un hueco para cargar un turno", "«Avisar a 3» ofrece el hueco a la lista de espera")),
        Paso("banknote", "La plata del negocio", "Cobrás cada turno desde la app y ves la caja del día.", listOf("Señas con Mercado Pago al reservar", "Membresías y cupones", "Reportes: ingresos, ocupación y horas muertas")),
        Paso("person.2", "Tus clientas y tu equipo", "Fichas con historial, notas y etiquetas. Horarios y días libres de cada profesional.", listOf("Penalidades para quien falta sin avisar", "Recordatorios de cumpleaños y «te extrañamos»")),
        Paso("sparkles", "Tu web y la IA", "Tu página de reservas se edita desde acá, y la IA trabaja para vos.", listOf("Que te reserven desde ChatGPT y Claude", "Manejá la agenda hablándole a tu asistente", "Logo, fotos, colores y plantilla")),
        Paso("bell.badge", "Todo te avisa", "Reservas, cancelaciones y huecos que se liberan, con botones para resolver sin abrir la app.", listOf("Widget con tu próximo turno", "La notificación con el turno en curso", "Todo lo demás, en la pestaña Más")),
    )
    val pagina = rememberPagerState { pasos.size }
    val alcance = rememberCoroutineScope()
    val ultima = pagina.currentPage == pasos.lastIndex
    Dialog(onDismissRequest = alTerminar, properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false)) {
        Column(Modifier.fillMaxSize().fondoPetroleo(C.petroleo).statusBarsPadding().navigationBarsPadding()) {
            HorizontalPager(pagina, Modifier.weight(1f)) { i ->
                val p = pasos[i]
                Column(Modifier.fillMaxSize().padding(horizontal = 28.dp), verticalArrangement = Arrangement.spacedBy(18.dp, Alignment.CenterVertically)) {
                    Box(Modifier.size(76.dp).clip(RoundedCornerShape(22.dp)).background(C.menta), contentAlignment = Alignment.Center) { Icono(p.simbolo, 34f, C.petroleoHondo) }
                    Texto(p.titulo, Tipo.lato(32f, Peso.Black), Color.White, tracking = (-0.8).sp)
                    Texto(p.texto, Tipo.lato(17f), Color.White.copy(alpha = 0.82f))
                    Column(Modifier.padding(top = 4.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        p.puntos.forEach { x ->
                            Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                                Icono("checkmark.circle.fill", 17f, C.menta)
                                Texto(x, Tipo.lato(15f, Peso.Bold), Color.White)
                            }
                        }
                    }
                    Spacer(Modifier.height(80.dp))
                }
            }
            Row(Modifier.fillMaxWidth().padding(bottom = 20.dp), horizontalArrangement = Arrangement.spacedBy(6.dp, Alignment.CenterHorizontally)) {
                pasos.indices.forEach { i ->
                    val ancho by animateDpAsState(if (i == pagina.currentPage) 20.dp else 6.dp, Movimiento.tocar(), label = "punto")
                    Box(Modifier.size(ancho, 6.dp).clip(CircleShape).background(if (i == pagina.currentPage) C.menta else Color.White.copy(alpha = 0.3f)))
                }
            }
            Column(Modifier.padding(start = 20.dp, end = 20.dp, bottom = 12.dp), verticalArrangement = Arrangement.spacedBy(8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                BotonMenta(if (ultima) "Empezar" else "Siguiente", { if (ultima) alTerminar() else alcance.launch { pagina.animateScrollToPage(pagina.currentPage + 1) } })
                Texto("Saltear", Tipo.lato(15f, Peso.Bold), Color.White.copy(alpha = 0.75f), Modifier.alpha(if (ultima) 0f else 1f).clickable(enabled = !ultima, onClick = alTerminar).padding(12.dp))
            }
        }
    }
}

/** Más › «Qué podés hacer»: el recorrido otra vez y cómo sumar el widget. */
@Composable
fun RecorridoOtraVez() {
    var tour by remember { mutableStateOf(false) }
    // Los pasos del widget y «El turno en curso» (Isla Dinámica en iOS) siguen cómo se hace en Android.
    PantallaMas("Qué podés hacer", "Todo lo que trae Tengo Libre", recargar = {}) {
        Box(Modifier.fillMaxWidth().tarjeta().clickable { tour = true }) { FilaLista("play.circle", "Ver el recorrido", "Un minuto, cinco pantallas") }
        Bloque("Sumar el widget", "Muestra tu próximo turno y lo que sigue.",
            { Renglon("1. Mantené apretada la pantalla de inicio", simbolo = "hand.tap") },
            { Renglon("2. Tocá «Widgets»", simbolo = "plus.app") },
            { Renglon("3. Buscá «Tengo Libre» y arrastralo", simbolo = "magnifyingglass") })
        Bloque("El turno en curso", "Cuando empieza un turno con la app abierta, queda en la notificación con el tiempo que falta y el botón «Atendido».",
            { Renglon("Turno en curso, siempre a la vista", simbolo = "timelapse") })
        Bloque("Avisos con botones", "Desplegá un aviso para confirmar, avisar a la lista o responder una reseña sin abrir la app.",
            { Renglon("Resolvé desde la notificación", simbolo = "bell.badge") })
    }
    if (tour) Recorrido { tour = false }
}
