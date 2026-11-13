package ar.com.tengolibre.app.nucleo

import android.content.Context
import android.os.Bundle
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.push.Push
import ar.com.tengolibre.app.widgets.Widgets
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de Nucleo/Actividad.swift. La Live Activity de iOS es, en Android, una notificación en curso con la cuenta
// regresiva y «Atendido» (en Android 16 se pide como Live Update: arriba en la barra y en la pantalla bloqueada).

/** El turno en curso: se prende cuando empieza uno (con la app abierta en Hoy) y le pasa el token de FCM a la API
 *  (`POST /v1/panel/turnos/{id}/actividad` con plataforma android): desde ahí el servidor la actualiza (quién
 *  sigue, si lo movieron) y la termina (atendido, cancelado, ausente) aunque la app esté cerrada. Con la app abierta
 *  también se actualiza desde acá. */
object ActividadTurno {
    private const val AVISO = 7001
    private val registrados = mutableSetOf<String>()
    private var actual: String? = null

    @Serializable private data class Registro(val token: String, val plataforma: String = "android", val entorno: String = "produccion")

    suspend fun sincronizar(turnos: List<Turno>, estado: Estado, ahora: Instant = Instant.now()) {
        val ctx = Preferencias.contexto ?: return
        Widgets.actualizar(ctx)
        val activos = turnos.filter { it.estado in listOf("pendiente", "confirmado") }.sortedBy { it.inicio }
        val t = activos.firstOrNull { !it.inicio.isAfter(ahora) && it.fin.isAfter(ahora) }
        if (t == null) { actual?.let { terminar(ctx, it) }; return }
        val sigue = activos.firstOrNull { !it.inicio.isBefore(t.fin) }?.let { "${Formato.horaDe(it.inicio)} · ${it.cliente.nombre.substringBefore(' ')}" }.orEmpty()
        mostrar(ctx, t.id, t.cliente.nombre, t.servicio, t.profesional, t.fin, sigue)
        if (t.id !in registrados) Push.token(ctx)?.let { token ->
            if (runCatching { estado.api.hacer("POST", "/v1/panel/turnos/${t.id}/actividad", Registro(token)) }.isSuccess) registrados.add(t.id)
        }
    }

    /** Lo que manda el servidor por FCM: {tipo: actividad, turno, fin, siguiente, terminar, cliente, servicio}. */
    fun desdePush(ctx: Context, d: Map<String, String>) {
        val turno = d["turno"] ?: return
        if (d["terminar"] == "1") { terminar(ctx, turno); return }
        val fin = d["fin"]?.let { runCatching { Instant.parse(it) }.getOrNull() } ?: return
        mostrar(ctx, turno, d["cliente"].orEmpty(), d["servicio"].orEmpty(), d["profesional"].orEmpty(), fin, d["siguiente"].orEmpty())
        Widgets.actualizar(ctx)
    }

    private fun mostrar(ctx: Context, turno: String, cliente: String, servicio: String, profesional: String, fin: Instant, siguiente: String) {
        if (!Push.hayPermiso(ctx)) return
        actual = turno
        val texto = listOf(servicio, profesional).filter { it.isNotEmpty() }.joinToString(" · ")
        val b = NotificationCompat.Builder(ctx, Push.CANAL_TURNO)
            .setSmallIcon(R.drawable.ic_aviso)
            .setColor(0xFF3CCC90.toInt())
            .setSubText("En curso")
            .setContentTitle(cliente)
            .setContentText(if (siguiente.isEmpty()) texto else "$texto\nSigue: $siguiente")
            .setStyle(NotificationCompat.BigTextStyle().bigText(if (siguiente.isEmpty()) texto else "$texto\nSigue: $siguiente"))
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setUsesChronometer(true)
            .setChronometerCountDown(true)
            .setWhen(fin.toEpochMilli())
            .setShowWhen(true)
            .setTimeoutAfter(maxOf(60_000L, fin.toEpochMilli() - System.currentTimeMillis() + 15 * 60_000L))
            .setCategory(NotificationCompat.CATEGORY_PROGRESS)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setContentIntent(Push.abrir(ctx, "turnos/$turno", null, null, AVISO))
            .addAction(0, "Atendido", Push.accion(ctx, "ATENDIDO", turno, null, AVISO))
            // Android 16: pedirla como Live Update (EXTRA_REQUEST_PROMOTED_ONGOING); antes se ignora.
            .addExtras(Bundle().apply { putBoolean("android.requestPromotedOngoing", true) })
        runCatching { NotificationManagerCompat.from(ctx).notify(AVISO, b.build()) }
    }

    fun terminar(ctx: Context, turno: String) {
        if (actual != null && actual != turno) return
        actual = null
        NotificationManagerCompat.from(ctx).cancel(AVISO)
        Widgets.actualizar(ctx)
    }
}
