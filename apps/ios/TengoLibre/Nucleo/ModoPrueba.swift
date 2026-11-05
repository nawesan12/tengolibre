#if DEBUG
import Foundation

/// Modo prueba (sólo Debug): con `-modoPrueba` la app arranca con sesión y una API simulada, para los
/// recorridos de XCUITest en GitHub Actions (TengoLibreUITests). `-modoCliente` entra como clienta y
/// `-conRecorrido` muestra el recorrido de la primera vez. Nunca llega a TestFlight (Release).
enum ModoPrueba {
    static var activo: Bool { ProcessInfo.processInfo.arguments.contains("-modoPrueba") }
    static var cliente: Bool { ProcessInfo.processInfo.arguments.contains("-modoCliente") }

    /// Deja la app como recién entrada: modo, permisos y recorrido.
    static func preparar() {
        guard activo else { return }
        let d = UserDefaults.standard
        if let b = Bundle.main.bundleIdentifier { d.removePersistentDomain(forName: b) }
        d.set(cliente ? "cliente" : "negocio", forKey: "modo")
        d.set(true, forKey: "permisosVistos")
        d.set(!ProcessInfo.processInfo.arguments.contains("-conRecorrido"), forKey: "recorridoVisto")
        d.set("n1", forKey: "negocio")
        if cliente { d.set(["ABC123"], forKey: "misTurnos") }
    }

    static func api() -> Api {
        let conf = URLSessionConfiguration.ephemeral
        conf.protocolClasses = [ApiDePrueba.self]
        return Api(base: URL(string: "https://api.prueba")!, token: "prueba", http: URLSession(configuration: conf))
    }
}

/// La API simulada: respuestas fijas por «MÉTODO ruta» (o sólo ruta para GET). Lo que no está: GET → 404,
/// lo demás → 200 con `{}`.
final class ApiDePrueba: URLProtocol {
    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let ruta = request.url?.path ?? ""
        let metodo = request.httpMethod ?? "GET"
        let r = Datos.respuestas()
        let cuerpo = r["\(metodo) \(ruta)"] ?? (metodo == "GET" ? r[ruta] : "{}")
        let codigo = cuerpo == nil ? 404 : 200
        let h = HTTPURLResponse(url: request.url!, statusCode: codigo, httpVersion: nil, headerFields: ["content-type": "application/json"])!
        client?.urlProtocol(self, didReceive: h, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: Data((cuerpo ?? #"{"status":404,"code":"no_existe","title":"No existe","detail":"No existe"}"#).utf8))
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}

    enum Datos {
        static func iso(_ min: Int) -> String { ISO8601DateFormatter().string(from: Date().addingTimeInterval(TimeInterval(min * 60))) }
        static func dia(_ dias: Int) -> String { Formato.diaISO(Date().addingTimeInterval(TimeInterval(dias * 86400))) }

        static func turno(_ id: String, _ cli: String, _ cid: String, _ srv: String, desde: Int, dura: Int, estado: String, origen: String = "web") -> String {
            """
            {"id": "\(id)", "codigo": "C\(id.uppercased())", "servicioId": "s1", "servicio": "\(srv)", "profesionalId": "p1", "profesional": "Carla",
             "cliente": {"id": "\(cid)", "nombre": "\(cli)", "telefono": "1144712290", "email": "\(cid)@prueba.test"},
             "inicio": "\(iso(desde))", "fin": "\(iso(desde + dura))", "precio": 18000, "estado": "\(estado)", "origen": "\(origen)", "nota": "",
             "venceEn": \(estado == "pendiente_pago" ? "\"\(iso(15))\"" : "null"), "creadoEn": "\(iso(-600))"}
            """
        }

        static func respuestas() -> [String: String] {
            let t1 = turno("t1", "Sofía Méndez", "c1", "Corte y brushing", desde: -10, dura: 45, estado: "confirmado")
            let t2 = turno("t2", "Martina López", "c2", "Color", desde: 90, dura: 60, estado: "pendiente_pago")
            let t3 = turno("t3", "Lucía Pérez", "c3", "Manicura", desde: 200, dura: 45, estado: "confirmado", origen: "panel")
            let t0 = turno("t0", "Ana Torres", "c4", "Corte", desde: -150, dura: 45, estado: "atendido")
            let horario = (0...6).map { "{\"dia\": \($0), \"desde\": \"00:00\", \"hasta\": \"23:59\"}" }.joined(separator: ", ")
            let pros = """
            [{"id": "p1", "nombre": "Carla Ruiz", "especialidad": "Colorista", "activo": true, "orden": 1, "servicios": ["s1", "s2"], "horario": [\(horario)],
              "email": "carla@prueba.test", "telefono": "", "bio": "", "miembroId": null, "foto": null},
             {"id": "p2", "nombre": "Abril Gómez", "especialidad": "Manicura", "activo": true, "orden": 2, "servicios": ["s2"], "horario": [\(horario)],
              "email": "", "telefono": "", "bio": "", "miembroId": null, "foto": null}]
            """
            let cliente = """
            {"id": "c1", "nombre": "Sofía Méndez", "telefono": "1144712290", "email": "c1@prueba.test", "creadoEn": "\(iso(-90000))", "visitas": 6,
             "ultimaVisita": "\(iso(-20000))", "proximoTurno": "\(iso(1440))", "gastado": 98000, "ausencias": 1, "penalizado": false,
             "etiquetas": [{"id": "e1", "nombre": "VIP", "color": "#3ccc90"}]}
            """
            let libres = (0..<6).map { i in "{\"hora\": \"\(16 + i / 2):\(i % 2 == 0 ? "00" : "30")\", \"inicio\": \"\(iso(300 + i * 30))\", \"profesionales\": [\"p1\"]}" }.joined(separator: ", ")
            let negocio = """
            {"id": "n1", "slug": "lumi", "nombre": "Lumi Estética", "descripcion": "", "direccion": "Thames 2240", "ciudad": "CABA", "telefono": "1144440000",
             "whatsapp": "1144440000", "instagram": "", "logo": null, "zonaHoraria": "America/Argentina/Buenos_Aires"}
            """
            return [
                "/v1/auth/persona": "{\"clerkId\": \"u1\", \"email\": \"laura@lumi.test\", \"nombre\": \"Laura Díaz\", \"foto\": \"\"}",
                "/v1/yo": "{\"negocios\": [{\"id\": \"n1\", \"slug\": \"lumi\", \"nombre\": \"Lumi Estética\", \"rol\": \"duenio\"}]}",
                "/v1/panel/acceso": "{\"bloqueado\": false, \"pruebaHasta\": \"\(iso(20000))\", \"diasDePrueba\": 14, \"conSuscripcion\": false}",
                "/v1/panel/campanita/sin-leer": "{\"sinLeer\": 2}",
                "/v1/panel/campanita": """
                {"sinLeer": 2, "notificaciones": [{"id": "x1", "tipo": "reserva", "titulo": "Nueva reserva", "texto": "Martina · hoy", "turnoId": "t1", "creadoEn": "\(iso(-3))", "nueva": true},
                 {"id": "x2", "tipo": "resena", "titulo": "Nueva reseña", "texto": "Lucía te dejó 5 estrellas", "turnoId": null, "creadoEn": "\(iso(-90))", "nueva": true}]}
                """,
                "/v1/panel/turnos": "{\"turnos\": [\(t0), \(t1), \(t2), \(t3)]}",
                "/v1/panel/turnos/t1": t1,
                "/v1/panel/turnos/t2": t2,
                "POST /v1/panel/turnos": turno("t9", "Sofía Méndez", "c1", "Corte y brushing", desde: 300, dura: 45, estado: "confirmado", origen: "panel"),
                "POST /v1/panel/turnos/t1/reprogramar": turno("t1", "Sofía Méndez", "c1", "Corte y brushing", desde: 1500, dura: 45, estado: "confirmado"),
                "/v1/panel/turnos/t1/caja": """
                {"turnoId": "t1", "cliente": "Sofía", "servicio": "Corte", "estadoTurno": "confirmado", "estado": "pendiente", "medio": null, "cobradoEn": null,
                 "lineas": [{"tipo": "servicio", "concepto": "Corte y brushing", "monto": 18000}], "total": 18000, "medios": ["efectivo", "transferencia"]}
                """,
                "/v1/panel/avisos": "{\"avisos\": []}",
                "/v1/panel/profesionales": "{\"profesionales\": \(pros)}",
                "/v1/panel/servicios": """
                {"servicios": [{"id": "s1", "nombre": "Corte y brushing", "duracion": 45, "precio": 18000, "activo": true, "categoriaId": null},
                 {"id": "s2", "nombre": "Manicura", "duracion": 45, "precio": 12000, "activo": true, "categoriaId": null}]}
                """,
                "/v1/panel/libres": "{\"horarios\": [\(libres)]}",
                "/v1/panel/espera": """
                {"espera": [{"id": "w1", "servicioId": "s1", "servicio": "Corte y brushing", "profesionalId": null, "profesional": null, "nombre": "Ana Torres",
                  "email": "", "telefono": "1166660000", "desde": "\(dia(0))", "hasta": "\(dia(10))", "avisadoEn": null, "creadoEn": "\(iso(-3000))", "momento": "cualquiera"}]}
                """,
                "/v1/panel/espera/ofertas": "{\"ofertas\": []}",
                "/v1/panel/espera/candidatos": "{\"espera\": []}",
                "/v1/panel/para-llamar": "{\"turnos\": []}",
                "/v1/panel/reprogramar": "{\"turnos\": []}",
                "/v1/panel/arranque": "{\"completo\": true, \"reservable\": true, \"pasos\": [{\"clave\": \"servicios\", \"titulo\": \"Cargá tus servicios\", \"listo\": true, \"obligatorio\": true}]}",
                "/v1/panel/resenas": """
                {"promedio": 4.8, "cantidad": 12, "distribucion": {"5": 10, "4": 2}, "resenas": [{"id": "r1", "turnoId": "t0", "nombre": "Lucía Pérez", "servicio": "Manicura",
                  "profesional": "Abril", "puntaje": 5, "comentario": "¡Divino todo! Volvería mil veces.", "respuesta": "", "respondidaEn": null, "visible": true, "creadoEn": "\(iso(-2000))"}]}
                """,
                "/v1/panel/dias-libres": "{\"pedidos\": []}",
                "/v1/panel/mercadopago": "{\"disponible\": true, \"conectada\": false, \"reautorizar\": false}",
                "/v1/panel/cobros": "{\"modo\": \"sin_cobro\", \"senaPorcentaje\": 30, \"senaFija\": 0}",
                "/v1/panel/pagos": "{\"pagos\": []}",
                "/v1/panel/clientes": "{\"clientes\": [\(cliente)], \"total\": 1}",
                "/v1/panel/clientes/c1": """
                {"id": "c1", "nombre": "Sofía Méndez", "telefono": "1144712290", "email": "c1@prueba.test", "creadoEn": "\(iso(-90000))", "visitas": 6,
                 "ultimaVisita": "\(iso(-20000))", "proximoTurno": null, "gastado": 98000, "ausencias": 1, "penalizado": false,
                 "etiquetas": [{"id": "e1", "nombre": "VIP", "color": "#3ccc90"}],
                 "turnos": [{"id": "t1", "codigo": "CT1", "servicio": "Corte", "profesional": "Carla", "inicio": "\(iso(-20000))", "fin": "\(iso(-19955))",
                   "precio": 18000, "estado": "atendido", "origen": "web", "nota": ""}], "penalidad": null}
                """,
                "/v1/panel/clientes/c1/notas": "{\"notas\": [{\"id\": \"n1\", \"texto\": \"Prefiere la tarde\", \"miembroId\": null, \"autor\": \"Laura\", \"creadoEn\": \"\(iso(-500))\"}]}",
                "/v1/panel/clientes/c1/penalidades": "{\"penalidades\": []}",
                "/v1/panel/etiquetas": "{\"etiquetas\": [{\"id\": \"e1\", \"nombre\": \"VIP\", \"color\": \"#3ccc90\", \"clientes\": 3}]}",
                "/v1/panel/membresias/planes": "{\"planes\": []}",
                "/v1/panel/membresias/socios": "{\"socios\": []}",
                "/v1/panel/cupones": "{\"cupones\": []}",
                "/v1/panel/recordatorios": "{\"recordatorios\": []}",
                "/v1/panel/ia/publico": "{\"activo\": false, \"reservasMes\": 0, \"ultimas\": [], \"url\": \"https://api.prueba/ia\"}",
                "/v1/panel/mcp": "{\"url\": \"https://api.prueba/mcp\", \"tokens\": [], \"permisos\": []}",
                "/v1/panel/calendarios": "{\"calendarios\": [{\"id\": null, \"profesionalId\": null, \"nombre\": \"Todo el negocio\", \"url\": null}]}",
                "/v1/panel/mi-web/galeria": "{\"fotos\": []}",
                "/v1/panel/negocio": negocio,
                "/v1/panel/mi-web": """
                {"rubro": "Estética", "provincia": "CABA", "acerca": "", "portada": null, "colorPrimario": "#183c44", "colorAcento": "#3ccc90", "email": "", "sitioWeb": "",
                 "facebook": "", "tiktok": "", "latitud": null, "longitud": null, "estilo": {"plantilla": "carta", "portada": "color", "fondo": "crema", "titulos": "grandes", "forma": "redondos"}}
                """,
                "/v1/panel/dominio": "{\"dominio\": null, \"destino\": \"tengolibre.com.ar\", \"ip\": \"187.41.102.190\"}",
                "/v1/panel/ajustes/whatsapp": "{\"disponible\": true, \"confirmacion\": false, \"hora\": \"10:00\", \"sinRespuestaHoras\": 4, \"recordatorio\": false}",
                "/v1/panel/caja": """
                {"fecha": "\(dia(0))", "porMedio": {"efectivo": 18000, "transferencia": 12000, "debito": 0, "credito": 0, "mercadopago": 0}, "caja": 30000, "senas": 5000,
                 "total": 35000, "productos": 0, "cobros": 2, "debe": 0, "movimientos": [{"turnoId": "t0", "cliente": "Ana Torres", "servicio": "Corte", "medio": "efectivo",
                 "total": 18000, "productos": 0, "cobradoEn": "\(iso(-100))"}]}
                """,
                "/v1/panel/reportes": """
                {"desde": "\(dia(-7))", "hasta": "\(dia(0))", "resumen": {"turnos": 42, "atendidos": 38, "ingresos": 684000, "ticketPromedio": 18000, "clientes": 31,
                 "tasaAusencias": 4.5, "ocupacion": 71.0}, "porEstado": {}, "porProfesional": [{"id": "p1", "nombre": "Carla Ruiz", "turnos": 25, "atendidos": 23, "ausentes": 1,
                 "ingresos": 414000, "minutosReservados": 1100}], "porServicio": [], "porDia": [], "clientesNuevos": 6, "clientesRecurrentes": 25}
                """,
                "/v1/panel/reportes/horas-muertas": "{\"peor\": {\"texto\": \"Los martes de 14 a 16 casi siempre quedan vacíos.\"}, \"sugerencia\": null}",
                "/v1/panel/push/preferencias": "{\"rol\": \"duenio\", \"porDefecto\": true, \"apagados\": [], \"tipos\": [{\"tipo\": \"reserva\", \"nombre\": \"Reservas nuevas\", \"activo\": true}]}",
                "/v1/panel/buscar": "{\"clientes\": [{\"id\": \"c1\", \"nombre\": \"Sofía Méndez\", \"telefono\": \"1144712290\", \"email\": \"\"}], \"turnos\": [], \"servicios\": []}",
                // Modo clienta: su turno, el lugar y el directorio.
                "/v1/t/ABC123": """
                {"id": "t7", "codigo": "ABC123", "servicioId": "s1", "servicio": "Corte y brushing", "profesionalId": "p1", "profesional": "Carla",
                 "inicio": "\(iso(1500))", "fin": "\(iso(1545))", "precio": 18000, "estado": "confirmado", "venceEn": null,
                 "negocio": \(negocio), "puedeCancelar": true, "cancelarHastaHoras": 24}
                """,
                "/v1/p/lumi": """
                {"negocio": \(negocio), "servicios": [{"id": "s1", "categoriaId": null, "nombre": "Corte y brushing", "descripcion": "", "duracion": 45, "precio": 18000,
                 "foto": null, "orden": 1, "activo": true}, {"id": "s2", "categoriaId": null, "nombre": "Manicura", "descripcion": "", "duracion": 45, "precio": 12000,
                 "foto": null, "orden": 2, "activo": true}], "profesionales": [{"id": "p1", "nombre": "Carla", "especialidad": "", "foto": null, "servicios": []}],
                 "diasVista": 30, "politicaCancelacion": "Si no podés venir, avisá 24 h antes.", "web": {"portada": null, "colorPrimario": "#183c44", "acerca": "", "rubro": "Estética"},
                 "resenas": {"promedio": 4.8, "cantidad": 12}}
                """,
                "/v1/p/lumi/dias": "{\"dias\": [" + (0..<7).map { "{\"fecha\": \"\(dia($0))\", \"libres\": \($0 == 0 ? 0 : 4)}" }.joined(separator: ", ") + "]}",
                "/v1/p/lumi/libres": "{\"horarios\": [\(libres)]}",
                "POST /v1/p/lumi/turnos": "{\"codigo\": \"NUEVO1\", \"inicio\": \"\(iso(1800))\"}",
                "/v1/t/NUEVO1": """
                {"id": "t8", "codigo": "NUEVO1", "servicioId": "s1", "servicio": "Corte y brushing", "profesionalId": "p1", "profesional": "Carla",
                 "inicio": "\(iso(1800))", "fin": "\(iso(1845))", "precio": 18000, "estado": "confirmado", "venceEn": null, "negocio": \(negocio), "puedeCancelar": true}
                """,
                "/v1/explorar": "{\"negocios\": [{\"slug\": \"lumi\", \"nombre\": \"Lumi Estética\", \"descripcion\": \"\", \"logo\": null, \"portada\": null, \"ciudad\": \"CABA\", \"provincia\": \"\", \"rubro\": \"Estética\", \"servicios\": [], \"desde\": 12000}]}",
                "/v1/explorar/rubros": "{\"rubros\": [\"Estética\", \"Peluquería\", \"Uñas\"]}"
            ]
        }
    }
}
#endif
