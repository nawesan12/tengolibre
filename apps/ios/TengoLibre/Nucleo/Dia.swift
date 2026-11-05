import Foundation

/// El día de cada profesional (firma «El día» del handoff): franjas de trabajo, lo ocupado y los
/// huecos libres. La misma cuenta que la web (apps/web/src/lib/panel/dia.ts): en minutos desde la
/// medianoche del negocio, con el horario semanal y los turnos (no mira bloqueos ni excepciones:
/// eso lo sabe /v1/panel/libres, que se usa al cargar un turno).
enum Dia {
    struct Tramo: Equatable, Hashable { var desde: Int; var hasta: Int; var largo: Int { hasta - desde } }

    struct Hueco: Identifiable, Equatable, Hashable {
        var profesionalId: String
        var profesional: String
        var desde: Int
        var hasta: Int
        var id: String { "\(profesionalId)-\(desde)" }
        var minutos: Int { hasta - desde }
    }

    struct DeProfesional: Identifiable, Equatable {
        var id: String
        var nombre: String
        var franjas: [Tramo]
        var ocupado: [(tramo: Tramo, turno: Turno)]
        var huecos: [Hueco]
        var libres: Int
        static func == (a: Self, b: Self) -> Bool { a.id == b.id && a.huecos == b.huecos && a.ocupado.map { $0.turno } == b.ocupado.map { $0.turno } }
    }

    /// Un hueco más corto que esto no se ofrece.
    static let minimo = 30

    static func aMin(_ h: String) -> Int {
        let p = h.split(separator: ":").compactMap { Int($0) }
        return (p.first ?? 0) * 60 + (p.count > 1 ? p[1] : 0)
    }

    static func hhmm(_ m: Int) -> String { String(format: "%d:%02d", m / 60, m % 60) }

    /// 0 = domingo, como la API y la web.
    static func diaDeSemana(_ fecha: Date) -> Int {
        Formato.calendario.component(.weekday, from: fecha) - 1
    }

    static func franjas(_ p: Profesional, _ fecha: Date) -> [Tramo] {
        let d = diaDeSemana(fecha)
        var out: [Tramo] = []
        for f in p.horario where f.dia == d {
            let t = Tramo(desde: aMin(f.desde), hasta: aMin(f.hasta))
            if t.hasta > t.desde { out.append(t) }
        }
        out.sort { $0.desde < $1.desde }
        return out
    }

    /// Resta los ocupados a las franjas: lo que queda libre desde `desde`.
    static func restar(_ franjas: [Tramo], _ ocupados: [Tramo], desde: Int = 0) -> [Tramo] {
        var out: [Tramo] = []
        let orden = ocupados.sorted { $0.desde < $1.desde }
        for f in franjas {
            var ini = max(f.desde, desde)
            for o in orden {
                if o.hasta <= ini || o.desde >= f.hasta { continue }
                if o.desde > ini { out.append(Tramo(desde: ini, hasta: min(o.desde, f.hasta))) }
                ini = max(ini, o.hasta)
                if ini >= f.hasta { break }
            }
            if ini < f.hasta { out.append(Tramo(desde: ini, hasta: f.hasta)) }
        }
        return out
    }

    /// El día de cada profesional. Con `ahora` (minutos), los huecos arrancan en el próximo cuarto de
    /// hora; con nil se cuenta el día entero (otro día que no es hoy).
    static func armar(_ profesionales: [Profesional], _ turnos: [Turno], fecha: Date, ahora: Int?) -> [DeProfesional] {
        let dia = Formato.diaISO(fecha)
        let delDia = turnos.filter { Formato.diaISO($0.inicio) == dia }
        return profesionales.map { p in
            let fr = franjas(p, fecha)
            let ocupado = delDia.filter { $0.profesionalId == p.id && $0.ocupa }
                .map { t -> (tramo: Tramo, turno: Turno) in
                    let d = Formato.minutosDelDia(t.inicio)
                    return (Tramo(desde: d, hasta: min(24 * 60, d + max(5, t.minutos))), t)
                }
                .sorted { $0.tramo.desde < $1.tramo.desde }
            let corte = ahora.map { Int((Double($0) / 15).rounded(.up)) * 15 } ?? 0
            let huecos = restar(fr, ocupado.map { $0.tramo }, desde: corte)
                .filter { $0.largo >= minimo }
                .map { Hueco(profesionalId: p.id, profesional: p.nombre, desde: $0.desde, hasta: $0.hasta) }
            return DeProfesional(id: p.id, nombre: p.nombre, franjas: fr, ocupado: ocupado, huecos: huecos,
                                 libres: huecos.reduce(0) { $0 + $1.minutos })
        }
    }

    /// De qué hora a qué hora va el día (para dibujar la franja).
    static func rango(_ dias: [DeProfesional], base: (Int, Int) = (9 * 60, 20 * 60)) -> (Int, Int) {
        let todo = dias.flatMap { d in d.franjas + d.ocupado.map { $0.tramo } }
        guard !todo.isEmpty else { return base }
        let desde = (todo.map(\.desde).min()! / 60) * 60
        let hasta = Int((Double(todo.map(\.hasta).max()!) / 60).rounded(.up)) * 60
        return (desde, max(hasta, desde + 60))
    }

    /// Quiénes entran: los activos (o con turnos ese día). Una profesional ve sólo lo suyo.
    static func profesionalesDelDia(_ ps: [Profesional], _ turnos: [Turno], rol: String, propio: String? = nil) -> [Profesional] {
        if rol == "profesional" {
            if let propio { return ps.filter { $0.id == propio } }
            let con = Set(turnos.compactMap(\.profesionalId))
            return ps.filter { con.contains($0.id) }
        }
        return ps.filter { p in p.activo || turnos.contains { $0.profesionalId == p.id } }
    }

    static func minutosAhora(_ d: Date = Date()) -> Int { Formato.minutosDelDia(d) }

    /// La fecha (en hora de Argentina) de un día AAAA-MM-DD a esos minutos.
    static func fecha(_ dia: Date, minutos: Int) -> Date {
        var c = Formato.calendario.dateComponents([.year, .month, .day], from: dia)
        c.hour = minutos / 60
        c.minute = minutos % 60
        return Formato.calendario.date(from: c) ?? dia
    }
}
