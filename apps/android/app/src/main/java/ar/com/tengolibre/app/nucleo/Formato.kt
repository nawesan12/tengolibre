package ar.com.tengolibre.app.nucleo

import kotlinx.serialization.KSerializer
import kotlinx.serialization.Serializable
import kotlinx.serialization.descriptors.PrimitiveKind
import kotlinx.serialization.descriptors.PrimitiveSerialDescriptor
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder
import java.text.DecimalFormat
import java.text.DecimalFormatSymbols
import java.time.DayOfWeek
import java.time.Duration
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit
import java.util.Locale

/** Montos, porcentajes y fechas como los escribe el handoff (clon de Nucleo/Formato.swift): `$ 184.350`
 *  (espacio fino que no corta el renglón, punto de miles, coma decimal), `12,4 %`, «hoy, 14:32». */
object Formato {
    /** Espacio fino sin corte (U+202F): entre `$` y el número, y antes de `%`. */
    const val fino = " "

    val argentina: Locale = Locale.forLanguageTag("es-AR")
    val zona: ZoneId = ZoneId.of("America/Argentina/Buenos_Aires")
    /** La semana arranca el lunes (firstWeekday = 2 en iOS). */
    val primerDia: DayOfWeek = DayOfWeek.MONDAY

    private val simbolos = DecimalFormatSymbols(argentina).apply { groupingSeparator = '.'; decimalSeparator = ',' }
    private val miles get() = DecimalFormat("#,##0", simbolos)
    private val decimal get() = DecimalFormat("#,##0.#", simbolos)

    /** Un número entero con punto de miles: 1.234.567. */
    fun numero(n: Int): String = miles.format(n)

    /** Pesos enteros (la API manda pesos sin centavos): `$ 24.020`, `−$ 2.400`. */
    fun pesos(n: Int): String = if (n < 0) "−$$fino${numero(-n)}" else "$$fino${numero(n)}"
    fun pesos(n: Double): String = pesos(Math.round(n).toInt())

    /** Un porcentaje con un decimal si hace falta: `12,4 %`, `73 %`. */
    fun porcentaje(p: Double): String = "${decimal.format(p)}$fino%"

    private fun formateador(patron: String) = DateTimeFormatter.ofPattern(patron, argentina).withZone(zona)
    private val hora = formateador("H:mm")
    private val diaLargo = formateador("EEEE d 'de' MMMM")
    private val diaCorto = formateador("d 'de' MMM")
    private val mes = formateador("MMMM")

    fun enArgentina(f: Instant): ZonedDateTime = f.atZone(zona)
    private fun mismoDia(a: Instant, b: Instant) = enArgentina(a).toLocalDate() == enArgentina(b).toLocalDate()

    /** «14:32». */
    fun horaDe(f: Instant): String = hora.format(f)
    /** «jueves 30 de octubre». */
    fun fechaLarga(f: Instant): String = diaLargo.format(f)
    /** «octubre». */
    fun mesDe(f: Instant): String = mes.format(f)

    /** «hoy», «ayer», o «28 de oct.». */
    fun dia(f: Instant, ahora: Instant = Instant.now()): String = when {
        mismoDia(f, ahora) -> "hoy"
        mismoDia(f, ahora.minus(1, ChronoUnit.DAYS)) -> "ayer"
        else -> diaCorto.format(f)
    }

    /** Título de sección por día: «Hoy», «Ayer», «Martes 28 de octubre». */
    fun seccion(f: Instant, ahora: Instant = Instant.now()): String = when {
        mismoDia(f, ahora) -> "Hoy"
        mismoDia(f, ahora.minus(1, ChronoUnit.DAYS)) -> "Ayer"
        else -> mayuscula(diaLargo.format(f))
    }

    /** «hoy, 14:32». */
    fun cuando(f: Instant, ahora: Instant = Instant.now()): String = "${dia(f, ahora)}, ${horaDe(f)}"

    /** «hace 3 h», «hace 12 min», «hace 2 días». */
    fun hace(f: Instant, ahora: Instant = Instant.now()): String {
        val s = maxOf(0L, Duration.between(f, ahora).seconds)
        if (s < 60) return "recién"
        if (s < 3600) return "hace ${s / 60}${fino}min"
        if (s < 86400) return "hace ${s / 3600}${fino}h"
        val d = s / 86400
        return if (d == 1L) "hace 1 día" else "hace $d días"
    }

    fun mayuscula(s: String): String = s.take(1).uppercase(argentina) + s.drop(1)

    /** Iniciales para los avatares: «Almacén Lumbre» → «AL». */
    fun iniciales(nombre: String): String {
        val p = nombre.split(' ', '-').filter { it.firstOrNull()?.isLetter() == true }
        val s = p.take(2).map { it.first().toString() }.joinToString("")
        return if (s.isEmpty()) "TL" else s.uppercase(argentina)
    }

    /** «1 producto», «5 productos». */
    fun cantidad(n: Int, singular: String, plural: String): String = "${numero(n)} ${if (n == 1) singular else plural}"

    private val cortas = setOf("a", "al", "y", "o", "u", "e", "de", "del", "el", "la", "los", "las", "en", "un", "una",
        "con", "por", "que", "se", "tu", "te", "su", "lo", "mi", "es", "no", "ya", "si", "ni")

    /** Une palabras cortas a la siguiente con un espacio duro, para que no queden solas al final de un renglón. */
    fun sinViudas(s: String): String {
        val palabras = s.split(" ")
        if (palabras.size <= 2) return s
        val r = StringBuilder()
        palabras.forEachIndexed { i, w ->
            r.append(w)
            if (i == palabras.size - 1) return@forEachIndexed
            // La última palabra va pegada a la anteúltima: nunca queda sola en el renglón final.
            val pegar = w.lowercase() in cortas || i == palabras.size - 2
            r.append(if (pegar) " " else " ")
        }
        return r.toString()
    }

    // Ayudas de fecha y texto propias de los turnos (extension Formato en Modelos.swift).

    /** «45 min», «1 h», «1 h 30». */
    fun duracion(minutos: Int): String {
        if (minutos < 60) return "$minutos min"
        val h = minutos / 60; val m = minutos % 60
        return if (m == 0) "$h h" else "$h h $m"
    }

    /** «45 min libres», «1 h 30 libre». */
    fun libres(minutos: Int): String = "${duracion(minutos)} ${if (minutos in 60..119) "libre" else "libres"}"

    /** El día en formato de la API (2026-11-04), en hora de Argentina. */
    fun diaISO(f: Instant): String = enArgentina(f).toLocalDate().toString()

    /** «lunes 2 de noviembre, 10:30». */
    fun diaYHora(f: Instant): String = "${fechaLarga(f)}, ${horaDe(f)}"

    /** Minutos desde la medianoche de Argentina. */
    fun minutosDelDia(f: Instant): Int = enArgentina(f).let { it.hour * 60 + it.minute }
}

object Fecha {
    private val fraccion = Regex("""\.\d+""")

    /** RFC 3339 con o sin fracción (Go manda nanosegundos), o una fecha sola (en hora de Argentina). */
    fun iso(s: String): Instant? {
        runCatching { return OffsetDateTime.parse(s).toInstant() }
        runCatching { return Instant.parse(s) }
        fraccion.find(s)?.let { m -> runCatching { return OffsetDateTime.parse(s.replaceRange(m.range, m.value.take(10))).toInstant() } }
        return runCatching { LocalDate.parse(s).atStartOfDay(Formato.zona).toInstant() }.getOrNull()
    }

    fun texto(f: Instant): String = DateTimeFormatter.ISO_INSTANT.format(f)
}

/** Las fechas de la API (Codable con Fecha.iso en iOS). */
object FechaSerializer : KSerializer<Instant> {
    override val descriptor = PrimitiveSerialDescriptor("Fecha", PrimitiveKind.STRING)
    override fun deserialize(decoder: Decoder): Instant = Fecha.iso(decoder.decodeString()) ?: Instant.EPOCH
    override fun serialize(encoder: Encoder, value: Instant) = encoder.encodeString(Fecha.texto(value))
}

typealias FechaApi = @Serializable(with = FechaSerializer::class) Instant
