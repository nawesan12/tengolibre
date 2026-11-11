package ar.com.tengolibre.app.app

import android.net.Uri
import ar.com.tengolibre.app.pantallas.mas.DestinoMas

/** Los links de tengolibre.com.ar que abren la app (App Links; clon de App/Vinculos.swift). Qué paths los declara
 *  `apps/pruebas/enlaces.json` (de ahí sale el assetlinks de la web y el manifest lo repite); acá, a qué pantalla
 *  lleva cada uno. Lo que la app no tiene se abre en la web dentro de la app, nunca queda en nada. */
object Vinculos {
    fun ruta(u: Uri): Ruta? {
        val host = u.host ?: return null
        if (host != "tengolibre.com.ar" && host != "www.tengolibre.com.ar") return null
        val texto = u.toString()
        if ("web" in u.queryParameterNames) return Ruta.web(texto)
        val p = u.pathSegments
        return when {
            p.firstOrNull() == "turno" && p.size > 1 -> Ruta.turnoCliente(p[1])
            p.firstOrNull() == "invitacion" && p.size > 1 -> Ruta.invitacion(p[1])
            p.firstOrNull() == "panel" -> panel(p.drop(1), texto)
            else -> Ruta.web(texto)
        }
    }

    /** `/panel/<sección>/<id>` → la pantalla de la app que hace lo mismo. */
    private fun panel(p: List<String>, u: String): Ruta {
        val id = p.getOrNull(1)
        return when (p.firstOrNull()) {
            null -> Ruta.hoy
            "turnos" -> when (id) { null -> Ruta.hoy; "nuevo" -> Ruta.nuevoTurno; else -> Ruta.turno(id) }
            "calendario" -> if (id == null) Ruta.agenda else Ruta.web(u)
            "lista-espera" -> Ruta.espera
            "clientes" -> Ruta.clientes
            "notificaciones" -> if (id == null) Ruta.avisos else Ruta.web(u)
            "buscar" -> if (id == null) Ruta.buscar else Ruta.web(u)
            "resenas" -> Ruta.resenas
            "suscripcion" -> Ruta.suscripcion
            "servicios" -> Ruta.mas(DestinoMas.servicios)
            "equipo" -> Ruta.mas(DestinoMas.equipo)
            "profesionales" -> Ruta.mas(if (id != null) DestinoMas.profesional(id) else DestinoMas.equipo)
            "pagos" -> Ruta.mas(DestinoMas.cobros)
            "membresias" -> Ruta.mas(DestinoMas.membresias)
            "cupones" -> Ruta.mas(DestinoMas.cupones)
            "referidos" -> Ruta.mas(DestinoMas.referidos)
            "reportes" -> if (id == null || id == "mensual") Ruta.mas(DestinoMas.reportes) else Ruta.web(u)
            "mi-web" -> Ruta.mas(DestinoMas.miWeb)
            "plantillas" -> Ruta.mas(DestinoMas.plantillas)
            "dominio" -> Ruta.mas(DestinoMas.dominio)
            "integraciones" -> Ruta.mas(DestinoMas.integraciones)
            "configuracion" -> Ruta.mas(DestinoMas.configuracion)
            "etiquetas" -> Ruta.mas(DestinoMas.etiquetas)
            "penalidades" -> Ruta.mas(DestinoMas.penalidades)
            "auditoria" -> if (id == null) Ruta.mas(DestinoMas.auditoria) else Ruta.web(u)
            "sucursales" -> Ruta.mas(DestinoMas.sucursales)
            else -> Ruta.web(u)
        }
    }
}

/** Cómo se escribe la ruta en `apps/pruebas/enlaces.json` («turno:42», «mas:servicios», «web-en-app»). */
val Ruta.clave: String get() = when (this) {
    is Ruta.turno -> "turno:$id"
    Ruta.hoy -> "hoy"
    Ruta.espera -> "espera"
    Ruta.avisos -> "avisos"
    Ruta.resenas -> "resenas"
    Ruta.nuevoTurno -> "nuevoTurno"
    Ruta.suscripcion -> "suscripcion"
    Ruta.caja -> "caja"
    Ruta.buscar -> "buscar"
    Ruta.agenda -> "agenda"
    is Ruta.turnoCliente -> "turnoCliente:$codigo"
    is Ruta.invitacion -> "invitacion:$token"
    Ruta.clientes -> "clientes"
    is Ruta.web -> "web-en-app"
    is Ruta.mas -> when (val d = destino) {
        is DestinoMas.profesional -> "mas:profesional:${d.id}"
        else -> "mas:" + d.toString()
    }
}
