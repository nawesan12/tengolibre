package ar.com.tengolibre.app.nucleo

import java.time.Instant
import kotlin.math.ceil

/** El día de cada profesional (clon de Nucleo/Dia.swift): franjas de trabajo, lo ocupado y los huecos libres.
 *  La misma cuenta que la web (apps/web/src/lib/panel/dia.ts), en minutos desde la medianoche del negocio. */
object Dia {
    data class Tramo(val desde: Int, val hasta: Int) { val largo get() = hasta - desde }

    data class Hueco(val profesionalId: String, val profesional: String, val desde: Int, val hasta: Int) {
        val id get() = "$profesionalId-$desde"
        val minutos get() = hasta - desde
    }

    data class Ocupado(val tramo: Tramo, val turno: Turno)

    data class DeProfesional(
        val id: String,
        val nombre: String,
        val franjas: List<Tramo>,
        val ocupado: List<Ocupado>,
        val huecos: List<Hueco>,
        val libres: Int,
    )

    /** Un hueco más corto que esto no se ofrece. */
    const val minimo = 30

    fun aMin(h: String): Int {
        val p = h.split(":").mapNotNull { it.toIntOrNull() }
        return (p.firstOrNull() ?: 0) * 60 + (if (p.size > 1) p[1] else 0)
    }

    fun hhmm(m: Int): String = "%d:%02d".format(m / 60, m % 60)

    /** 0 = domingo, como la API y la web. */
    fun diaDeSemana(fecha: Instant): Int = Formato.enArgentina(fecha).dayOfWeek.value % 7

    fun franjas(p: Profesional, fecha: Instant): List<Tramo> {
        val d = diaDeSemana(fecha)
        return p.horario.filter { it.dia == d }.map { Tramo(aMin(it.desde), aMin(it.hasta)) }.filter { it.hasta > it.desde }.sortedBy { it.desde }
    }

    /** Resta los ocupados a las franjas: lo que queda libre desde `desde`. */
    fun restar(franjas: List<Tramo>, ocupados: List<Tramo>, desde: Int = 0): List<Tramo> {
        val out = mutableListOf<Tramo>()
        val orden = ocupados.sortedBy { it.desde }
        for (f in franjas) {
            var ini = maxOf(f.desde, desde)
            for (o in orden) {
                if (o.hasta <= ini || o.desde >= f.hasta) continue
                if (o.desde > ini) out.add(Tramo(ini, minOf(o.desde, f.hasta)))
                ini = maxOf(ini, o.hasta)
                if (ini >= f.hasta) break
            }
            if (ini < f.hasta) out.add(Tramo(ini, f.hasta))
        }
        return out
    }

    /** El día de cada profesional. Con `ahora` (minutos), los huecos arrancan en el próximo cuarto de hora;
     *  con null se cuenta el día entero (otro día que no es hoy). */
    fun armar(profesionales: List<Profesional>, turnos: List<Turno>, fecha: Instant, ahora: Int?): List<DeProfesional> {
        val dia = Formato.diaISO(fecha)
        val delDia = turnos.filter { Formato.diaISO(it.inicio) == dia }
        return profesionales.map { p ->
            val fr = franjas(p, fecha)
            val ocupado = delDia.filter { it.profesionalId == p.id && it.ocupa }
                .map { t -> val d = Formato.minutosDelDia(t.inicio); Ocupado(Tramo(d, minOf(24 * 60, d + maxOf(5, t.minutos))), t) }
                .sortedBy { it.tramo.desde }
            val corte = ahora?.let { (ceil(it / 15.0) * 15).toInt() } ?: 0
            val huecos = restar(fr, ocupado.map { it.tramo }, desde = corte).filter { it.largo >= minimo }
                .map { Hueco(p.id, p.nombre, it.desde, it.hasta) }
            DeProfesional(p.id, p.nombre, fr, ocupado, huecos, huecos.sumOf { it.minutos })
        }
    }

    /** De qué hora a qué hora va el día (para dibujar la franja). */
    fun rango(dias: List<DeProfesional>, base: Pair<Int, Int> = 9 * 60 to 20 * 60): Pair<Int, Int> {
        val todo = dias.flatMap { d -> d.franjas + d.ocupado.map { it.tramo } }
        if (todo.isEmpty()) return base
        val desde = (todo.minOf { it.desde } / 60) * 60
        val hasta = (ceil(todo.maxOf { it.hasta } / 60.0) * 60).toInt()
        return desde to maxOf(hasta, desde + 60)
    }

    /** Quiénes entran: los activos (o con turnos ese día). Una profesional ve sólo lo suyo. */
    fun profesionalesDelDia(ps: List<Profesional>, turnos: List<Turno>, rol: String, propio: String? = null): List<Profesional> {
        if (rol == "profesional") {
            if (propio != null) return ps.filter { it.id == propio }
            val con = turnos.mapNotNull { it.profesionalId }.toSet()
            return ps.filter { it.id in con }
        }
        return ps.filter { p -> p.activo || turnos.any { it.profesionalId == p.id } }
    }

    fun minutosAhora(d: Instant = Instant.now()): Int = Formato.minutosDelDia(d)

    /** La fecha (en hora de Argentina) de ese día a esos minutos. */
    fun fecha(dia: Instant, minutos: Int): Instant =
        Formato.enArgentina(dia).toLocalDate().atTime(minutos / 60, minutos % 60).atZone(Formato.zona).toInstant()
}
