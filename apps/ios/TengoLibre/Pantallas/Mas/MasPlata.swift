import StoreKit
import SwiftUI

struct ConexionMP: Codable, Equatable { var disponible: Bool; var conectada: Bool; var desde: Date?; var venceEn: Date?; var reautorizar: Bool; var ultimoError: String?; var cuenta: String? }
struct ConfigCobro: Codable, Equatable { var modo: String; var senaPorcentaje: Int; var senaFija: Int }
struct PagoSena: Codable, Identifiable, Equatable, Hashable {
    var id: String; var turnoId: String; var codigo: String; var servicio: String; var cliente: String; var inicio: Date; var turnoEstado: String
    var monto: Int; var modo: String; var estado: String; var devuelto: Int; var venceEn: Date?; var vencido: Bool; var creadoEn: Date
}
struct Cupon: Codable, Identifiable, Equatable, Hashable {
    var id: String; var codigo: String; var tipo: String; var valor: Int; var minimo: Int?; var maxUsos: Int?; var usos: Int
    var desde: Date?; var hasta: Date?; var activo: Bool; var referidoId: String?; var creadoEn: Date?
    var texto: String { tipo == "porcentaje" ? "\(valor) % off" : "\(Formato.pesos(valor)) off" }
}
struct ResumenCaja: Codable, Equatable {
    struct Mov: Codable, Equatable, Hashable { var turnoId: String; var cliente: String; var servicio: String; var medio: String; var total: Int; var productos: Int; var cobradoEn: Date }
    var fecha: String; var porMedio: [String: Int]; var caja: Int; var senas: Int; var total: Int; var productos: Int; var cobros: Int; var debe: Int; var movimientos: [Mov]
}

let mediosPago: [String: String] = ["efectivo": "Efectivo", "transferencia": "Transferencia", "mercadopago": "QR de Mercado Pago", "debito": "Débito", "credito": "Crédito", "despues": "Debe"]

// MARK: - Pagos y señas

struct Cobros: View {
    @Environment(Estado.self) private var estado
    @Environment(\.openURL) private var abrir
    @State private var mp: ConexionMP?
    @State private var config = ConfigCobro(modo: "sin_cobro", senaPorcentaje: 30, senaFija: 0)
    @State private var pagos: [PagoSena] = []
    @State private var cargando = true
    @State private var error = ""
    @State private var devolver: PagoSena?

    var body: some View {
        PantallaMas(titulo: "Pagos y señas", subtitulo: mp?.conectada == true ? "Mercado Pago conectado" : "Sin Mercado Pago", cargando: cargando, error: error, recargar: cargar) {
            Bloque(titulo: "Mercado Pago", pie: "Las señas van directo a tu cuenta. Tengo Libre no cobra comisión.") {
                if let mp, mp.conectada {
                    Renglon(titulo: mp.cuenta ?? "Cuenta conectada", detalle: mp.desde.map { "Desde el \(Formato.dia($0))" } ?? "", simbolo: "checkmark.seal.fill")
                    if mp.reautorizar {
                        Button { Task { await conectar() } } label: { Renglon(titulo: "Volver a autorizar", detalle: mp.ultimoError ?? "Mercado Pago pide que la conectes de nuevo", simbolo: "exclamationmark.triangle") }.buttonStyle(.plain)
                    }
                    Button { Task { await desconectar() } } label: { Renglon(titulo: "Desconectar", simbolo: "xmark.circle") }.buttonStyle(.plain).foregroundStyle(C.error)
                } else {
                    Button { Task { await conectar() } } label: {
                        HStack(spacing: 10) {
                            Image("LogoMercadoPago").resizable().scaledToFit().frame(width: 28, height: 28)
                            Text("Conectar Mercado Pago").font(Tipo.lato(16, .bold))
                        }
                        .frame(maxWidth: .infinity, minHeight: 56)
                    }
                    .buttonStyle(.plain).disabled(mp?.disponible == false)
                }
            }
            Bloque(titulo: "Cobro al reservar") {
                Renglon(titulo: "Qué se cobra") {
                    Picker("Modo", selection: $config.modo) {
                        Text("Nada").tag("sin_cobro")
                        Text("Seña en %").tag("porcentaje")
                        Text("Seña fija").tag("fija")
                        Text("El total").tag("total")
                    }
                    .tint(C.mentaTexto)
                }
                if config.modo == "porcentaje" { Numerito(titulo: "Porcentaje", valor: $config.senaPorcentaje, rango: 5...100, paso: 5) { "\($0) %" } }
                if config.modo == "fija" { RenglonPesos(titulo: "Monto", valor: $config.senaFija) }
                BotonGuardar(accion: guardarConfig).padding(12)
            }
            .disabled(mp?.conectada != true)
            if !pagos.isEmpty {
                Bloque(titulo: "Últimos pagos") {
                    ForEach(pagos.prefix(40)) { p in
                        Renglon(titulo: "\(p.cliente) · \(Formato.pesos(p.monto))", detalle: "\(p.servicio) · \(Formato.diaYHora(p.inicio)) · \(estadoPago(p))") {
                            if p.estado == "aprobado" && p.devuelto < p.monto {
                                Button("Devolver") { devolver = p }.buttonStyle(BotonTexto(color: C.error))
                            }
                        }
                    }
                }
            }
        }
        .task(id: estado.version) { await cargar() }
        .confirmationDialog("¿Devolver \(devolver.map { Formato.pesos($0.monto - $0.devuelto) } ?? "") a \(devolver?.cliente ?? "")?", isPresented: Binding(get: { devolver != nil }, set: { if !$0 { devolver = nil } }), titleVisibility: .visible) {
            Button("Devolver y cancelar el turno", role: .destructive) { if let p = devolver { Task { await hacerDevolucion(p) } } }
        }
    }

    private func estadoPago(_ p: PagoSena) -> String {
        if p.devuelto > 0 { return p.devuelto >= p.monto ? "devuelto" : "devuelto en parte" }
        return ["aprobado": "pagado", "pendiente": p.vencido ? "venció" : "esperando", "rechazado": "rechazado"][p.estado] ?? p.estado
    }

    private struct Pagos: Decodable { var pagos: [PagoSena] }
    private struct Url: Decodable { var url: String }

    private func cargar() async {
        do {
            mp = try await estado.api.get("/v1/panel/mercadopago")
            if let c: ConfigCobro = try? await estado.api.get("/v1/panel/cobros") { config = c }
            pagos = (try? await estado.api.get("/v1/panel/pagos") as Pagos)?.pagos ?? []
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
        cargando = false
    }

    private func conectar() async {
        do {
            let r: Url = try await estado.api.enviar("POST", "/v1/panel/mercadopago/conectar", [String: String]())
            if let u = URL(string: r.url) { abrir(u) }
        } catch { estado.avisar(error) }
    }

    private func desconectar() async {
        await estado.intentar("Desconectamos Mercado Pago") { try await estado.api.hacer("DELETE", "/v1/panel/mercadopago") }
        await cargar()
    }

    private func guardarConfig() async {
        await estado.intentar("Guardamos cómo cobrás") { try await estado.api.hacer("PUT", "/v1/panel/cobros", config) }
    }

    private func hacerDevolucion(_ p: PagoSena) async {
        await estado.intentar("Devolvimos el pago y cancelamos el turno") { try await estado.api.hacer("POST", "/v1/panel/pagos/\(p.id)/devolver", [String: Int]()) }
        await cargar()
    }
}

// MARK: - Caja

struct CajaDelDia: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var dia = Date()
    @State private var r: ResumenCaja?
    @State private var error = ""

    var body: some View {
        PantallaMas(titulo: "Caja", subtitulo: Formato.fechaLarga(dia), cargando: r == nil && error.isEmpty, error: error, recargar: cargar) {
            HStack {
                Button { cambiar(-1) } label: { Image(systemName: "chevron.left").frame(width: 44, height: 44) }.accessibilityLabel("Día anterior")
                Spacer()
                DatePicker("", selection: $dia, in: ...Date(), displayedComponents: .date).labelsHidden().colorScheme(.dark)
                Spacer()
                Button { cambiar(1) } label: { Image(systemName: "chevron.right").frame(width: 44, height: 44) }.accessibilityLabel("Día siguiente")
                    .disabled(Formato.diaISO(dia) >= Formato.diaISO(Date()))
            }
            .foregroundStyle(.white)
        } contenido: {
            if let r {
                VStack(alignment: .leading, spacing: 4) {
                    Text("TOTAL DEL DÍA").rotulo()
                    Text(Formato.pesos(r.total)).font(Tipo.lato(40, .black)).monospacedDigit().numeroAnimado(r.total)
                    Text("\(Formato.pesos(r.caja)) en el local · \(Formato.pesos(r.senas)) de señas").font(Tipo.lato(14)).foregroundStyle(C.sub)
                }
                .foregroundStyle(C.tinta).padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
                if r.total > 0 {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("CÓMO TE PAGARON").rotulo()
                        GraficoDona(partes: ["efectivo", "transferencia", "mercadopago", "debito", "credito"].enumerated().map { i, m in
                            GraficoDona.Parte(nombre: mediosPago[m] ?? m, valor: r.porMedio[m] ?? 0, color: Paleta.color(i))
                        } + (r.senas > 0 ? [GraficoDona.Parte(nombre: "Señas online", valor: r.senas, color: Paleta.color(5))] : []),
                                    centro: Formato.pesosCortos(r.total), pie: "en total")
                    }
                    .padding(16).tarjeta()
                    .id(r.fecha)
                    .entrada(1)
                }
                Bloque(titulo: "Por medio de pago") {
                    ForEach(["efectivo", "transferencia", "mercadopago", "debito", "credito"], id: \.self) { m in
                        Renglon(titulo: mediosPago[m] ?? m) { Text(Formato.pesos(r.porMedio[m] ?? 0)).font(Tipo.lato(15, .bold)).monospacedDigit() }
                    }
                    if r.productos > 0 { Renglon(titulo: "De eso, productos") { Text(Formato.pesos(r.productos)).monospacedDigit() } }
                    if r.debe > 0 { Renglon(titulo: "Quedaron debiendo", simbolo: "exclamationmark.circle") { Text(Formato.pesos(r.debe)).font(Tipo.lato(15, .bold)).foregroundStyle(C.ausenteTinta).monospacedDigit() } }
                }
                if !r.movimientos.isEmpty {
                    Bloque(titulo: "Cobros (\(r.cobros))") {
                        ForEach(r.movimientos, id: \.self) { m in
                            Button {
                                Task { if let t: Turno = try? await estado.api.get("/v1/panel/turnos/\(m.turnoId)") { nav.turno = t } }
                            } label: {
                                Renglon(titulo: m.cliente, detalle: "\(m.servicio) · \(mediosPago[m.medio] ?? m.medio) · \(Formato.horaDe(m.cobradoEn))") {
                                    Text(Formato.pesos(m.total)).font(Tipo.lato(15, .bold)).monospacedDigit()
                                }
                            }
                            .buttonStyle(.plain)
                        }
                    }
                } else {
                    Text("Ese día no hubo cobros en el local.").font(Tipo.lato(14)).foregroundStyle(C.sub)
                }
            }
        }
        .task(id: "\(estado.version)-\(Formato.diaISO(dia))") { await cargar() }
    }

    private func cambiar(_ d: Int) { if let x = Formato.calendario.date(byAdding: .day, value: d, to: dia), x <= Date() { dia = x } }

    private func cargar() async {
        do { r = try await estado.api.get("/v1/panel/caja", query: [URLQueryItem(name: "fecha", value: Formato.diaISO(dia))]); error = "" }
        catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Membresías

struct Membresias: View {
    @Environment(Estado.self) private var estado
    @State private var planes: [PlanMembresia] = []
    @State private var socios: [Socio] = []
    @State private var servicios: [Servicio] = []
    @State private var tab = "activa"
    @State private var cargando = true
    @State private var plan: PlanMembresia?
    @State private var nuevoPlan = false
    @State private var nuevoSocio = false
    @State private var socio: Socio?

    var body: some View {
        PantallaMas(titulo: "Membresías", subtitulo: "\(socios.filter { $0.estado == "activa" }.count) socias activas",
                    acciones: [AccionCabecera(simbolo: "person.badge.plus", etiqueta: "Sumar socia") { nuevoSocio = true }],
                    cargando: cargando, recargar: cargar) {
            Segmentado(opciones: [("activa", "Activas"), ("vencida", "Vencidas"), ("planes", "Planes")], elegido: $tab)
        } contenido: {
            if tab == "planes" {
                ForEach(planes.sorted { $0.orden < $1.orden }) { p in
                    Button { plan = p } label: {
                        VStack(alignment: .leading, spacing: 6) {
                            HStack {
                                Text(p.nombre).font(Tipo.lato(17, .bold))
                                Spacer()
                                Text("\(Formato.pesos(p.precio))/mes").font(Tipo.lato(15, .bold)).monospacedDigit()
                            }
                            Text("\(Formato.cantidad(p.visitas, "visita", "visitas")) cada \(p.duracionDias) días\(p.prioridadDias > 0 ? " · reserva \(p.prioridadDias) días antes" : "")").font(Tipo.lato(13)).foregroundStyle(C.sub)
                            Text("\(socios.filter { $0.planId == p.id && $0.estado == "activa" }.count)\(p.topeSocios.map { " de \($0)" } ?? "") socias\(p.activo ? "" : " · pausado")").font(Tipo.lato(12, .bold)).foregroundStyle(C.mentaTexto)
                        }
                        .foregroundStyle(C.tinta).padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
                    }
                    .buttonStyle(.plain)
                }
                Button { nuevoPlan = true } label: { Label("Crear un plan", systemImage: "plus") }.buttonStyle(BotonContorno())
            } else {
                let l = socios.filter { $0.estado == tab }
                if l.isEmpty {
                    VacioQueEnsena(rayas: "Sin socias", titulo: tab == "activa" ? "Todavía no hay socias" : "No hay vencidas", texto: "Las membresías son abonos mensuales con visitas incluidas. Se venden desde tu página o las cargás acá.") { EmptyView() }
                } else {
                    Bloque {
                        ForEach(l) { s in
                            Button { socio = s } label: {
                                Renglon(titulo: s.cliente, detalle: "\(s.plan) · quedan \(s.quedan) de \(s.visitasDelPlan) · vence \(Fecha.iso(s.hasta).map { Formato.dia($0) } ?? s.hasta)") {
                                    Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(C.sub)
                                }
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
        }
        .task(id: estado.version) { await cargar() }
        .sheet(item: $plan) { p in EditarPlan(plan: p, servicios: servicios) { await cargar() } }
        .sheet(isPresented: $nuevoPlan) {
            EditarPlan(plan: PlanMembresia(id: "", nombre: "", descripcion: "", precio: 0, visitas: 4, duracionDias: 30, prioridadDias: 0, beneficios: [], topeSocios: nil, servicios: [], activo: true, orden: planes.count + 1), servicios: servicios) { await cargar() }
        }
        .sheet(isPresented: $nuevoSocio) { NuevaSocia(planes: planes.filter(\.activo)) { await cargar() } }
        .sheet(item: $socio) { s in AccionesSocia(socio: s) { await cargar() } }
    }

    private struct Ps: Decodable { var planes: [PlanMembresia] }
    private struct Ss: Decodable { var socios: [Socio] }
    private struct Sv: Decodable { var servicios: [Servicio] }

    private func cargar() async {
        planes = (try? await estado.api.get("/v1/panel/membresias/planes") as Ps)?.planes ?? planes
        socios = (try? await estado.api.get("/v1/panel/membresias/socios", query: [URLQueryItem(name: "estado", value: "todas")]) as Ss)?.socios ?? socios
        servicios = (try? await estado.api.get("/v1/panel/servicios") as Sv)?.servicios ?? servicios
        cargando = false
    }
}

struct EditarPlan: View {
    @State var plan: PlanMembresia
    var servicios: [Servicio]
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var beneficios = ""
    @State private var conTope = false
    @State private var tope = 20

    var body: some View {
        HojaEdicion(titulo: plan.id.isEmpty ? "Plan nuevo" : "Editar plan", habilitado: plan.nombre.count >= 2,
                    borrar: plan.id.isEmpty ? nil : { await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/membresias/planes/\(plan.id)") }; await listo() },
                    accion: guardar) {
            Bloque {
                RenglonCampo(titulo: "Nombre", texto: $plan.nombre, placeholder: "Club del brushing")
                RenglonCampo(titulo: "Descripción", texto: $plan.descripcion, multilinea: true)
                RenglonPesos(titulo: "Precio por mes", valor: $plan.precio)
                Numerito(titulo: "Visitas incluidas", valor: $plan.visitas, rango: 1...60)
                Numerito(titulo: "Dura", valor: $plan.duracionDias, rango: 7...365, paso: 1) { "\($0) días" }
                Numerito(titulo: "Reserva antes que el resto", valor: $plan.prioridadDias, rango: 0...30) { $0 == 0 ? "No" : "\($0) días" }
                RenglonInterruptor(titulo: "Cupo de socias", activo: $conTope)
                if conTope { Numerito(titulo: "Máximo", valor: $tope, rango: 1...999) }
                RenglonCampo(titulo: "Beneficios (uno por renglón)", texto: $beneficios, multilinea: true)
                RenglonInterruptor(titulo: "A la venta", activo: $plan.activo)
            }
            Bloque(titulo: "Servicios que cubre") {
                ForEach(servicios.filter(\.activo)) { s in
                    RenglonInterruptor(titulo: s.nombre, activo: Binding(get: { plan.servicios.contains(s.id) }, set: { v in
                        if v { plan.servicios.append(s.id) } else { plan.servicios.removeAll { $0 == s.id } }
                    }))
                }
            }
        }
        .onAppear { beneficios = plan.beneficios.joined(separator: "\n"); conTope = plan.topeSocios != nil; tope = plan.topeSocios ?? 20 }
    }

    private struct Cuerpo: Encodable {
        var nombre: String; var descripcion: String; var precio: Int; var visitas: Int; var duracionDias: Int; var prioridadDias: Int
        var beneficios: [String]; var topeSocios: Int?; var servicios: [String]; var activo: Bool
    }

    private func guardar() async -> Bool {
        let c = Cuerpo(nombre: plan.nombre, descripcion: plan.descripcion, precio: plan.precio, visitas: plan.visitas, duracionDias: plan.duracionDias, prioridadDias: plan.prioridadDias,
                       beneficios: beneficios.split(separator: "\n").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty },
                       topeSocios: conTope ? tope : nil, servicios: plan.servicios, activo: plan.activo)
        let ok = await estado.intentar {
            if plan.id.isEmpty { try await estado.api.hacer("POST", "/v1/panel/membresias/planes", c) }
            else { try await estado.api.hacer("PATCH", "/v1/panel/membresias/planes/\(plan.id)", c) }
        }
        if ok { await listo() }
        return ok
    }
}

struct NuevaSocia: View {
    var planes: [PlanMembresia]
    var nombreInicial = ""
    var telefonoInicial = ""
    var emailInicial = ""
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var planId = ""
    @State private var nombre = ""
    @State private var telefono = ""
    @State private var email = ""
    @State private var nota = ""
    @State private var contactos = false

    var body: some View {
        HojaEdicion(titulo: "Sumar socia", guardar: "Sumar", habilitado: !planId.isEmpty && nombre.count >= 2, accion: {
            let ok = await estado.intentar("Sumamos a \(nombre)") {
                try await estado.api.hacer("POST", "/v1/panel/membresias/socios", ["planId": planId, "nombre": nombre, "telefono": telefono, "email": email, "nota": nota])
            }
            if ok { await listo() }
            return ok
        }) {
            Bloque {
                Renglon(titulo: "Plan") {
                    Picker("Plan", selection: $planId) { ForEach(planes) { Text($0.nombre).tag($0.id) } }.tint(C.mentaTexto)
                }
                RenglonCampo(titulo: "Nombre", texto: $nombre)
                RenglonCampo(titulo: "Teléfono", texto: $telefono, teclado: .phonePad)
                RenglonCampo(titulo: "Email", texto: $email, teclado: .emailAddress)
                RenglonCampo(titulo: "Nota", texto: $nota)
            }
            Button { contactos = true } label: { Label("Traer de Contactos", systemImage: "person.crop.circle") }.buttonStyle(BotonTexto())
        }
        .onAppear {
            planId = planes.first?.id ?? ""
            if nombre.isEmpty { nombre = nombreInicial; telefono = telefonoInicial; email = emailInicial }
        }
        .sheet(isPresented: $contactos) { SelectorContacto { n, t, e in nombre = n; telefono = t; email = e }.ignoresSafeArea() }
    }
}

struct AccionesSocia: View {
    var socio: Socio
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var ajustar = 1
    @State private var motivo = ""

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(socio.cliente).font(Tipo.lato(22, .black))
                        Text("\(socio.plan) · quedan \(socio.quedan) de \(socio.visitasDelPlan) · vence \(Fecha.iso(socio.hasta).map { Formato.dia($0) } ?? socio.hasta)").font(Tipo.lato(14)).foregroundStyle(C.sub)
                    }
                    Bloque(titulo: "Ajustar visitas") {
                        Numerito(titulo: ajustar >= 0 ? "Sumar" : "Restar", valor: $ajustar, rango: -20...20) { $0 > 0 ? "+\($0)" : "\($0)" }
                        RenglonCampo(titulo: "Motivo", texto: $motivo, placeholder: "Le regalamos una, se descontó mal…")
                        BotonGuardar(texto: "Ajustar", habilitado: ajustar != 0) { await patch(["ajustar": .numero(Double(ajustar)), "motivo": .texto(motivo)], ajustar > 0 ? "Sumamos las visitas" : "Restamos las visitas") }.padding(12)
                    }
                    Button { Task { await patch(["renovar": .booleano(true)], "Renovamos la membresía desde hoy") } } label: { Label("Renovar desde hoy", systemImage: "arrow.clockwise") }
                        .buttonStyle(BotonContorno(alto: 48, tam: 16))
                    if socio.estado == "activa" {
                        Button(role: .destructive) { Task { await patch(["estado": .texto("cancelada")], "Cancelamos la membresía") } } label: { Label("Cancelar la membresía", systemImage: "xmark.circle") }
                            .buttonStyle(BotonContorno(alto: 48, tam: 16)).foregroundStyle(C.error)
                    } else {
                        Button { Task { await patch(["estado": .texto("activa")], "Reactivamos la membresía") } } label: { Label("Reactivar", systemImage: "checkmark.circle") }
                            .buttonStyle(BotonContorno(alto: 48, tam: 16))
                    }
                }
                .padding(16)
            }
            .background(C.fondo)
            .navigationTitle("Socia").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("Listo") { cerrar() } } }
        }
        .presentationDetents([.medium, .large])
    }

    private func patch(_ c: [String: ValorJSON], _ ok: String) async {
        if await estado.intentar(ok, { try await estado.api.hacer("PATCH", "/v1/panel/membresias/socios/\(socio.id)", c) }) {
            await listo()
            cerrar()
        }
    }
}

// MARK: - Cupones

struct Cupones: View {
    @Environment(Estado.self) private var estado
    @State private var cupones: [Cupon] = []
    @State private var cargando = true
    @State private var editando: Cupon?
    @State private var nuevo = false

    var body: some View {
        PantallaMas(titulo: "Cupones", subtitulo: "\(cupones.filter(\.activo).count) activos",
                    acciones: [AccionCabecera(simbolo: "plus", etiqueta: "Crear cupón") { nuevo = true }], cargando: cargando, recargar: cargar) {
            let l = cupones.filter { $0.referidoId == nil }
            if l.isEmpty {
                VacioQueEnsena(rayas: "Sin cupones", titulo: "Creá tu primer cupón", texto: "Un código que tus clientas escriben al reservar para tener un descuento.") {
                    Button("Crear un cupón") { nuevo = true }.buttonStyle(BotonMenta(alto: 52, conSombra: false))
                }
            } else {
                Bloque {
                    ForEach(l) { c in
                        Button { editando = c } label: {
                            Renglon(titulo: c.codigo, detalle: "\(c.texto) · \(c.usos)\(c.maxUsos.map { " de \($0)" } ?? "") usos\(c.hasta.map { " · hasta el \(Formato.dia($0))" } ?? "")") {
                                Toggle("", isOn: Binding(get: { c.activo }, set: { v in Task { await activar(c, v) } })).labelsHidden().tint(C.menta)
                            }
                            .font(Tipo.lato(16)).monospacedDigit()
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
        .task(id: estado.version) { await cargar() }
        .sheet(item: $editando) { c in EditarCupon(cupon: c) { await cargar() } }
        .sheet(isPresented: $nuevo) { EditarCupon(cupon: Cupon(id: "", codigo: "", tipo: "porcentaje", valor: 10, usos: 0, activo: true)) { await cargar() } }
    }

    private struct Cs: Decodable { var cupones: [Cupon] }
    private func cargar() async { cupones = (try? await estado.api.get("/v1/panel/cupones") as Cs)?.cupones ?? cupones; cargando = false }
    private func activar(_ c: Cupon, _ v: Bool) async {
        await estado.intentar(v ? "Activamos el cupón" : "Pausamos el cupón") { try await estado.api.hacer("PATCH", "/v1/panel/cupones/\(c.id)", ["activo": v]) }
        await cargar()
    }
}

struct EditarCupon: View {
    @State var cupon: Cupon
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var conMinimo = false
    @State private var minimo = 0
    @State private var conMax = false
    @State private var maxUsos = 50
    @State private var conHasta = false
    @State private var hasta = Date().addingTimeInterval(30 * 86400)

    private struct Cuerpo: Encodable { var codigo: String?; var tipo: String; var valor: Int; var minimo: Int?; var maxUsos: Int?; var hasta: Date?; var activo: Bool }

    var body: some View {
        HojaEdicion(titulo: cupon.id.isEmpty ? "Cupón nuevo" : cupon.codigo, habilitado: cupon.codigo.count >= 3,
                    borrar: cupon.id.isEmpty ? nil : { await estado.intentar("Sacamos el cupón") { try await estado.api.hacer("DELETE", "/v1/panel/cupones/\(cupon.id)") }; await listo() },
                    accion: guardar) {
            Bloque {
                if cupon.id.isEmpty {
                    RenglonCampo(titulo: "Código", texto: Binding(get: { cupon.codigo }, set: { cupon.codigo = $0.uppercased().filter { $0.isLetter || $0.isNumber } }), placeholder: "VERANO10")
                }
                Renglon(titulo: "Tipo") {
                    Picker("Tipo", selection: $cupon.tipo) { Text("Porcentaje").tag("porcentaje"); Text("Monto fijo").tag("fijo") }.pickerStyle(.segmented).frame(maxWidth: 200)
                }
                if cupon.tipo == "porcentaje" { Numerito(titulo: "Descuento", valor: $cupon.valor, rango: 1...100, paso: 5) { "\($0) %" } }
                else { RenglonPesos(titulo: "Descuento", valor: $cupon.valor) }
                RenglonInterruptor(titulo: "Compra mínima", activo: $conMinimo)
                if conMinimo { RenglonPesos(titulo: "Mínimo", valor: $minimo) }
                RenglonInterruptor(titulo: "Límite de usos", activo: $conMax)
                if conMax { Numerito(titulo: "Usos", valor: $maxUsos, rango: 1...9999) }
                RenglonInterruptor(titulo: "Vence", activo: $conHasta)
                if conHasta { DatePicker("Hasta", selection: $hasta, in: Date()..., displayedComponents: .date).padding(.horizontal, 16).frame(minHeight: 52).tint(C.mentaTexto) }
                RenglonInterruptor(titulo: "Activo", activo: $cupon.activo)
            }
        }
        .onAppear {
            conMinimo = cupon.minimo != nil; minimo = cupon.minimo ?? 0
            conMax = cupon.maxUsos != nil; maxUsos = cupon.maxUsos ?? 50
            conHasta = cupon.hasta != nil; hasta = cupon.hasta ?? hasta
        }
    }

    private func guardar() async -> Bool {
        let c = Cuerpo(codigo: cupon.id.isEmpty ? cupon.codigo : nil, tipo: cupon.tipo, valor: cupon.valor, minimo: conMinimo ? minimo : nil, maxUsos: conMax ? maxUsos : nil,
                       hasta: conHasta ? Formato.calendario.date(bySettingHour: 23, minute: 59, second: 0, of: hasta) : nil, activo: cupon.activo)
        let ok = await estado.intentar(cupon.id.isEmpty ? "Creamos el cupón" : "Guardamos el cupón") {
            if cupon.id.isEmpty { try await estado.api.hacer("POST", "/v1/panel/cupones", c) }
            else { try await estado.api.hacer("PATCH", "/v1/panel/cupones/\(cupon.id)", c) }
        }
        if ok { await listo() }
        return ok
    }
}

// MARK: - Referidos

struct Referidos: View {
    struct Ajustes: Codable { var activo: Bool; var tipo: String?; var valor: Int?; var tope: Int? }
    struct Referido: Codable, Identifiable { var id: String; var codigo: String; var cliente: String; var usos: Int; var creadoEn: Date }
    @Environment(Estado.self) private var estado
    @State private var a = Ajustes(activo: false, tipo: "porcentaje", valor: 10, tope: nil)
    @State private var lista: [Referido] = []
    @State private var cargando = true

    var body: some View {
        PantallaMas(titulo: "Referidos", subtitulo: a.activo ? "Activo" : "Apagado", cargando: cargando, recargar: cargar) {
            Bloque(pie: "Cada clienta recibe un código para compartir. Quien reserva con él tiene el descuento, y ella también en su próximo turno.") {
                RenglonInterruptor(titulo: "Programa de referidos", activo: $a.activo)
                if a.activo {
                    Renglon(titulo: "Premio") {
                        Picker("Tipo", selection: Binding(get: { a.tipo ?? "porcentaje" }, set: { a.tipo = $0 })) { Text("Porcentaje").tag("porcentaje"); Text("Monto fijo").tag("fijo") }
                            .pickerStyle(.segmented).frame(maxWidth: 200)
                    }
                    if a.tipo == "fijo" { RenglonPesos(titulo: "Descuento", valor: Binding(get: { a.valor ?? 0 }, set: { a.valor = $0 })) }
                    else { Numerito(titulo: "Descuento", valor: Binding(get: { a.valor ?? 10 }, set: { a.valor = $0 }), rango: 1...100, paso: 5) { "\($0) %" } }
                    Numerito(titulo: "Usos por código", valor: Binding(get: { a.tope ?? 0 }, set: { a.tope = $0 == 0 ? nil : $0 }), rango: 0...100) { $0 == 0 ? "Sin tope" : "\($0)" }
                }
                BotonGuardar { await estado.intentar("Guardamos los referidos") { try await estado.api.hacer("PUT", "/v1/panel/referidos/config", a) } }.padding(12)
            }
            if !lista.isEmpty {
                Bloque(titulo: "Códigos") {
                    ForEach(lista) { r in Renglon(titulo: r.cliente, detalle: r.codigo) { Text(Formato.cantidad(r.usos, "uso", "usos")).font(Tipo.lato(14, .bold)) } }
                }
            }
        }
        .task { await cargar() }
    }

    private struct Rs: Decodable { var referidos: [Referido] }
    private func cargar() async {
        if let x: Ajustes = try? await estado.api.get("/v1/panel/referidos/config") { a = x }
        lista = (try? await estado.api.get("/v1/panel/referidos") as Rs)?.referidos ?? []
        cargando = false
    }
}

// MARK: - Penalidades y ausencias

struct Penalidades: View {
    struct AjustesAusencias: Codable { var umbralAusencias: Int; var diasPenalidad: Int; var marcarAusentes: Bool }
    struct Ausencias: Codable { var total: Int; var ultimos30: Int; var penalidadesVigentes: Int; var reincidentes: Int }
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var lista: [Penalidad] = []
    @State private var aj = AjustesAusencias(umbralAusencias: 2, diasPenalidad: 30, marcarAusentes: false)
    @State private var aus: Ausencias?
    @State private var todas = false
    @State private var cargando = true

    var body: some View {
        PantallaMas(titulo: "Penalidades", subtitulo: aus.map { "\($0.ultimos30) ausencias en 30 días · \($0.penalidadesVigentes) vigentes" } ?? "", cargando: cargando, recargar: cargar) {
            Bloque(titulo: "Automático", pie: "Si alguien falta sin avisar esa cantidad de veces, no puede reservar por la web durante esos días.") {
                Numerito(titulo: "Penalizar desde", valor: $aj.umbralAusencias, rango: 0...10) { $0 == 0 ? "Nunca" : "\($0) faltas" }
                Numerito(titulo: "Durante", valor: $aj.diasPenalidad, rango: 1...365, paso: 5) { "\($0) días" }
                RenglonInterruptor(titulo: "Marcar ausentes solos", detalle: "Los turnos sin marcar pasan a «No vino» al terminar el día", activo: $aj.marcarAusentes)
                BotonGuardar { await estado.intentar("Guardamos los ajustes") { try await estado.api.hacer("PUT", "/v1/panel/ausencias/ajustes", aj) } }.padding(12)
            }
            Button { Task { await marcar() } } label: { Label("Marcar ausentes los turnos pasados sin marcar", systemImage: "person.slash") }
                .buttonStyle(BotonContorno(alto: 48, tam: 15))
            Toggle("Ver también las levantadas", isOn: $todas).tint(C.menta).font(Tipo.lato(15)).padding(.horizontal, 4)
            if lista.isEmpty {
                Text("No hay penalidades \(todas ? "" : "vigentes").").font(Tipo.lato(14)).foregroundStyle(C.sub)
            } else {
                Bloque {
                    ForEach(lista) { p in
                        Renglon(titulo: p.cliente, detalle: "\(p.vigente ? "Hasta el \(Formato.dia(p.hasta))" : "Levantada") · \(Formato.cantidad(p.ausencias, "ausencia", "ausencias"))") {
                            if p.vigente {
                                Button("Levantar") { Task { await estado.intentar("Levantamos la penalidad") { try await estado.api.hacer("POST", "/v1/panel/penalidades/\(p.id)/levantar") }; await cargar() } }
                                    .buttonStyle(BotonTexto())
                            }
                        }
                        .contentShape(Rectangle())
                        .onTapGesture { nav.pestana = .clientes; nav.pilaClientes.append(DestinoClientes.ficha(p.clienteId)) }
                    }
                }
            }
        }
        .task(id: "\(estado.version)-\(todas)") { await cargar() }
    }

    private struct Ps: Decodable { var penalidades: [Penalidad] }
    private struct Marcados: Decodable { var marcados: Int }

    private func cargar() async {
        var q: [URLQueryItem] = []
        if todas { q.append(URLQueryItem(name: "todas", value: "1")) }
        lista = (try? await estado.api.get("/v1/panel/penalidades", query: q) as Ps)?.penalidades ?? lista
        if let a: AjustesAusencias = try? await estado.api.get("/v1/panel/ausencias/ajustes") { aj = a }
        aus = try? await estado.api.get("/v1/panel/ausencias")
        cargando = false
    }

    private func marcar() async {
        do {
            let r: Marcados = try await estado.api.enviar("POST", "/v1/panel/ausencias/marcar", [String: String]())
            estado.avisar(r.marcados == 0 ? "No había turnos para marcar" : "Marcamos \(Formato.cantidad(r.marcados, "ausente", "ausentes"))")
            estado.version += 1
        } catch { estado.avisar(error) }
    }
}

// MARK: - Etiquetas

struct Etiquetas: View {
    @Environment(Estado.self) private var estado
    @State private var lista: [EtiquetaConUso] = []
    @State private var nombre = ""
    @State private var color = "#3ccc90"
    @State private var cargando = true
    @State private var editando: EtiquetaConUso?

    static let colores = ["#3ccc90", "#183c44", "#e8a317", "#d9534f", "#7a5cfa", "#2f80ed", "#ec4899", "#8b5e34"]

    var body: some View {
        PantallaMas(titulo: "Etiquetas", subtitulo: Formato.cantidad(lista.count, "etiqueta", "etiquetas"), cargando: cargando, recargar: cargar) {
            Bloque(pie: "Sirven para separar clientas (VIP, nueva, alérgica) y filtrarlas en Clientes.") {
                ForEach(lista) { e in
                    Button { editando = e } label: {
                        Renglon(titulo: e.nombre, detalle: Formato.cantidad(e.clientes, "clienta", "clientas")) {
                            Circle().fill(Color(hexTexto: e.color) ?? C.chip).frame(width: 18, height: 18)
                        }
                    }
                    .buttonStyle(.plain)
                }
                VStack(spacing: 10) {
                    HStack {
                        TextField("Nueva etiqueta", text: $nombre).font(Tipo.lato(16))
                        Button("Crear") { Task { await crear() } }.buttonStyle(BotonMentaChico()).disabled(nombre.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                    SelectorColor(color: $color)
                }
                .padding(16)
            }
        }
        .task { await cargar() }
        .sheet(item: $editando) { e in EditarEtiqueta(e: e) { await cargar() } }
    }

    private struct Es: Decodable { var etiquetas: [EtiquetaConUso] }
    private func cargar() async { lista = (try? await estado.api.get("/v1/panel/etiquetas") as Es)?.etiquetas ?? lista; cargando = false }
    private func crear() async {
        await estado.intentar { try await estado.api.hacer("POST", "/v1/panel/etiquetas", ["nombre": nombre, "color": color]) }
        nombre = ""
        await cargar()
    }
}

struct SelectorColor: View {
    @Binding var color: String
    var body: some View {
        HStack(spacing: 10) {
            ForEach(Etiquetas.colores, id: \.self) { c in
                Button { color = c } label: {
                    Circle().fill(Color(hexTexto: c) ?? .gray).frame(width: 28, height: 28)
                        .overlay(Circle().strokeBorder(C.tinta, lineWidth: color == c ? 3 : 0))
                }
                .buttonStyle(.plain).accessibilityLabel("Color \(c)")
            }
        }
    }
}

struct EditarEtiqueta: View {
    @State var e: EtiquetaConUso
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    var body: some View {
        HojaEdicion(titulo: "Etiqueta", habilitado: !e.nombre.isEmpty,
                    borrar: { await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/etiquetas/\(e.id)") }; await listo() },
                    accion: {
                        let ok = await estado.intentar { try await estado.api.hacer("PATCH", "/v1/panel/etiquetas/\(e.id)", ["nombre": e.nombre, "color": e.color]) }
                        if ok { await listo() }
                        return ok
                    }) {
            Bloque {
                RenglonCampo(titulo: "Nombre", texto: $e.nombre)
                SelectorColor(color: $e.color).padding(16)
            }
        }
        .presentationDetents([.medium])
    }
}

// MARK: - Suscripción

/// Suscripción: se adapta a dónde se contrató. Desde el iPhone se paga con la App Store (StoreKit 2,
/// se administra en Ajustes del iPhone); desde la web, con Mercado Pago (se maneja en la web); desde
/// Android, con Google Play (se administra allá, y acá no se compra otra). Sin
/// suscripción, los planes de la App Store con su precio, la renovación automática y los términos.
struct SuscripcionVista: View {
    struct PlanVenta: Codable, Hashable { var clave: String; var nombre: String; var precio: Int }
    struct Sus: Codable { var id: String; var plan: String; var estado: String; var monto: Int; var proximoCobro: Date?; var ultimoCobro: Date?; var graciaHasta: Date?; var canceladaEn: Date?; var origen: String? }
    struct EstadoS: Codable { var bloqueado: Bool; var pruebaHasta: Date?; var diasDePrueba: Int; var conSuscripcion: Bool; var plan: String; var nombrePlan: String; var catalogo: [PlanVenta]; var suscripcion: Sus? }
    @Environment(Estado.self) private var estado
    @State private var s: EstadoS?
    @State private var error = ""
    @State private var productos: [Product] = []
    @State private var cargoTienda = false
    @State private var comprando: String?
    @State private var administrar = false
    @State private var restaurando = false
    @State private var elegido = ""

    /// Qué trae cada plan (las mismas filas que la web).
    private let filas: [(String, String, String)] = [
        ("Turnos online y agenda", "Sí", "Sí"), ("Recordatorios por mail", "Sí", "Sí"), ("Profesionales", "Hasta 2", "Sin límite"),
        ("Señas y pagos con Mercado Pago", "No", "Sí"), ("Membresías, cupones y referidos", "No", "Sí"), ("Reportes", "Básicos", "Completos"),
        ("Dominio propio y plantillas", "No", "Sí"), ("Integraciones con IA", "No", "Sí"), ("Sucursales", "No", "Sí")
    ]

    private var vigente: Sus? { s?.suscripcion.flatMap { ["autorizada", "pausada"].contains($0.estado) ? $0 : nil } }

    var body: some View {
        PantallaMas(titulo: "Suscripción", subtitulo: subtitulo, cargando: s == nil && error.isEmpty, error: error, recargar: cargar) {
            if let s {
                if let v = vigente {
                    switch v.origen {
                    case "apple": deApple(v, s)
                    case "google": deGoogle(v, s)
                    default: deMercadoPago(v, s)
                    }
                } else {
                    estadoSinPlan(s)
                    if estado.esDuena { planesTienda } else {
                        Text("La suscripción la elige la dueña o el dueño del negocio.").font(Tipo.lato(14)).foregroundStyle(C.sub)
                    }
                }
            }
        }
        .task(id: estado.version) { await cargar() }
        .manageSubscriptionsSheet(isPresented: $administrar)
        .onChange(of: administrar) { _, abierto in if !abierto { Task { await cargar(); await estado.refrescar() } } }
    }

    private var subtitulo: String {
        guard let s else { return "" }
        if let v = vigente {
            let medio = v.origen == "apple" ? "App Store" : v.origen == "google" ? "Google Play" : "Mercado Pago"
            return "Plan \(s.nombrePlan) · \(medio)"
        }
        return s.bloqueado ? "La prueba terminó" : "Prueba gratis · quedan \(s.diasDePrueba) días"
    }

    // MARK: Contratada desde el iPhone

    @ViewBuilder private func deApple(_ v: Sus, _ s: EstadoS) -> some View {
        Bloque(titulo: "Tu plan") {
            Renglon(titulo: "Plan \(s.nombrePlan)", detalle: v.estado == "pausada" ? "La App Store no pudo cobrar" : "Se paga con la App Store", simbolo: "apple.logo")
            if let p = v.proximoCobro {
                Renglon(titulo: v.canceladaEn == nil ? "Se renueva" : "Termina", detalle: Formato.fechaLarga(p), simbolo: "calendar")
            }
            if v.estado == "pausada", let g = v.graciaHasta {
                Renglon(titulo: "Revisá tu medio de pago en Apple", detalle: "Todo sigue andando hasta el \(Formato.dia(g))", simbolo: "exclamationmark.triangle")
            }
        }
        if estado.esDuena {
            Button { administrar = true } label: { Label("Cambiar o cancelar en la App Store", systemImage: "gearshape") }
                .buttonStyle(BotonContorno(alto: 52, tam: 16))
            botonRestaurar
        }
        Text("Se renueva sola cada mes. Para cambiar de plan o cancelar: Ajustes › tu nombre › Suscripciones › Tengo Libre, o el botón de arriba.")
            .font(Tipo.lato(13)).foregroundStyle(C.sub)
    }

    // MARK: Contratada desde Android

    /// Se pagó con Google Play: acá no se compra ni se administra; se maneja donde se contrató.
    @ViewBuilder private func deGoogle(_ v: Sus, _ s: EstadoS) -> some View {
        Bloque(titulo: "Tu plan") {
            Renglon(titulo: "Plan \(s.nombrePlan)", detalle: v.estado == "pausada" ? "Google Play no pudo cobrar" : "Se paga con Google Play, desde Android", simbolo: "creditcard")
            if let p = v.proximoCobro {
                Renglon(titulo: v.canceladaEn == nil ? "Se renueva" : "Termina", detalle: Formato.fechaLarga(p), simbolo: "calendar")
            }
        }
        Text("La contrataste en un teléfono Android: el plan y la cancelación se administran donde la contrataste, en Google Play.")
            .font(Tipo.lato(14)).foregroundStyle(C.sub)
    }

    // MARK: Contratada desde la web

    @ViewBuilder private func deMercadoPago(_ v: Sus, _ s: EstadoS) -> some View {
        Bloque(titulo: "Tu plan") {
            Renglon(titulo: "Plan \(s.nombrePlan)", detalle: "Se paga con Mercado Pago, desde la web", simbolo: "creditcard")
            if let p = v.proximoCobro, v.estado == "autorizada" {
                Renglon(titulo: "Próximo cobro", detalle: "\(Formato.fechaLarga(p)) · \(Formato.pesos(v.monto))", simbolo: "calendar")
            }
            if v.estado == "pausada" {
                Renglon(titulo: v.canceladaEn == nil ? "Mercado Pago no pudo cobrar" : "Cancelada", detalle: v.graciaHasta.map { "Todo sigue andando hasta el \(Formato.dia($0))" } ?? "", simbolo: "exclamationmark.triangle")
            }
        }
        Text("La contrataste desde la web: el plan, la tarjeta y la cancelación se manejan desde el panel en la compu, en Suscripción.")
            .font(Tipo.lato(14)).foregroundStyle(C.sub)
    }

    // MARK: Sin suscripción

    @ViewBuilder private func estadoSinPlan(_ s: EstadoS) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(s.bloqueado ? "Tu prueba terminó" : "Estás en la prueba gratis").font(Tipo.lato(22, .black))
            Text(s.bloqueado ? "Podés ver todo, pero para cargar o cambiar turnos hace falta un plan. Tu página sigue tomando reservas."
                 : "Hasta el \(s.pruebaHasta.map { Formato.fechaLarga($0) } ?? "") tenés todo el plan Completo.")
                .font(Tipo.lato(15)).foregroundStyle(C.sub)
            if s.suscripcion?.estado == "cancelada", s.suscripcion?.origen == "apple" {
                Text("Tu suscripción de la App Store venció o se canceló.").font(Tipo.lato(14, .bold)).foregroundStyle(C.ausenteTinta)
            }
        }
        .foregroundStyle(C.tinta).padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
    }

    @ViewBuilder private var planesTienda: some View {
        if !cargoTienda {
            ForEach(0..<2, id: \.self) { _ in Esqueleto(alto: 96, radio: 18) }
            Esqueleto(alto: 260, radio: 20)
        } else if productos.isEmpty {
            VacioQueEnsena(rayas: "Pronto", titulo: "Los planes no cargaron", texto: "La App Store no respondió. Probá de nuevo en un rato.") {
                Button("Reintentar") { Task { await cargarTienda() } }.buttonStyle(BotonContorno(alto: 48, tam: 15))
            }
        } else {
            Text("1 · ELEGÍ TU PLAN").rotulo().padding(.horizontal, 4)
            ForEach(productos, id: \.id) { p in opcionPlan(p) }
            if let p = elegidoProducto { incluye(p) }
            Text("2 · TU SUSCRIPCIÓN").rotulo().padding(.horizontal, 4).padding(.top, 6)
            if let p = elegidoProducto { carrito(p) }
            botonRestaurar
            legales
        }
    }

    private static let descripcion: [String: String] = [
        "inicial": "Para empezar: turnos online, agenda y recordatorios, hasta 2 profesionales.",
        "completo": "Todo Tengo Libre: señas, membresías, reportes, IA y sin límite de equipo."
    ]

    private var elegidoProducto: Product? {
        productos.first { $0.id == elegido } ?? productos.first { Compras.productos[$0.id] == "completo" } ?? productos.first
    }

    private func opcionPlan(_ p: Product) -> some View {
        let clave = Compras.productos[p.id] ?? ""
        let on = elegidoProducto?.id == p.id
        return Button {
            withAnimation(Movimiento.tocar) { elegido = p.id }
            Haptico.seleccion()
        } label: {
            HStack(alignment: .center, spacing: 12) {
                Image(systemName: on ? "largecircle.fill.circle" : "circle").font(.system(size: 24))
                    .foregroundStyle(on ? C.mentaTexto : C.sub)
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 8) {
                        Text(p.displayName.isEmpty ? Formato.mayuscula(clave) : p.displayName).font(Tipo.lato(19, .black)).tracking(-0.4)
                        if clave == "completo" {
                            Text("RECOMENDADO").font(Tipo.lato(10, .black)).tracking(0.8).padding(.horizontal, 7).frame(height: 20)
                                .foregroundStyle(C.petroleoHondo).background(C.menta, in: Capsule())
                        }
                    }
                    Text(Self.descripcion[clave] ?? p.description).font(Tipo.lato(13)).foregroundStyle(C.sub).multilineTextAlignment(.leading)
                }
                Spacer(minLength: 4)
                VStack(alignment: .trailing, spacing: 0) {
                    Text(p.displayPrice).font(Tipo.lato(20, .black)).monospacedDigit()
                    Text("por mes").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
            }
            .foregroundStyle(C.tinta).padding(16)
            .background(C.tarjeta, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).strokeBorder(on ? C.menta : C.linea, lineWidth: 2))
            .shadow(color: on ? Color(hex: 0x17805A, alfa: 0.25) : .clear, radius: 14, y: 10)
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(on ? .isSelected : [])
    }

    private func incluye(_ p: Product) -> some View {
        let col = Compras.productos[p.id] == "completo" ? 2 : 1
        return VStack(alignment: .leading, spacing: 7) {
            Text("QUÉ INCLUYE").rotulo()
            ForEach(filas, id: \.0) { f in
                let v = col == 1 ? f.1 : f.2
                HStack(spacing: 8) {
                    Image(systemName: v == "No" ? "minus.circle" : "checkmark.circle.fill").foregroundStyle(v == "No" ? C.linea : C.mentaTexto)
                    Text(f.0).font(Tipo.lato(14)).foregroundStyle(v == "No" ? C.sub : C.tinta).strikethrough(v == "No", color: C.sub)
                    Spacer(minLength: 0)
                    if v != "Sí" && v != "No" { Text(v).font(Tipo.lato(13, .bold)).foregroundStyle(C.sub) }
                }
            }
        }
        .padding(16).frame(maxWidth: .infinity, alignment: .leading)
        .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .animation(Movimiento.tocar, value: p.id)
    }

    /// El «carrito»: el plan, cuándo y cuánto se cobra, el medio de pago y el total de hoy.
    private func carrito(_ p: Product) -> some View {
        let nombre = p.displayName.isEmpty ? Formato.mayuscula(Compras.productos[p.id] ?? "") : p.displayName
        let dia = Formato.calendario.component(.day, from: Date())
        let quedanPrueba = (s?.pruebaHasta).map { $0 > Date() } ?? false
        return VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 12) {
                Image(systemName: "crown.fill").font(.system(size: 18, weight: .bold)).foregroundStyle(C.petroleoHondo)
                    .frame(width: 40, height: 40).background(C.menta, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                VStack(alignment: .leading, spacing: 2) {
                    Text("Plan \(nombre)").font(Tipo.lato(16, .bold))
                    Text("Mensual · se renueva sola · cancelás cuando quieras").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
                Spacer(minLength: 4)
                Text(p.displayPrice).font(Tipo.lato(17, .black)).monospacedDigit()
            }
            .padding(.bottom, 14)
            Troquel(vertical: false)
            VStack(spacing: 9) {
                fila("Primer cobro", "hoy, al confirmar")
                fila("Después", "\(p.displayPrice) cada mes, el día \(dia)")
                fila("Medio de pago", "tu cuenta de Apple")
                if quedanPrueba {
                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: "info.circle").foregroundStyle(C.pagoTinta)
                        Text("Te quedan \(s?.diasDePrueba ?? 0) días de prueba. Si te suscribís desde el iPhone, Apple cobra hoy.")
                            .font(Tipo.lato(12)).foregroundStyle(C.pagoTinta)
                    }
                    .padding(10).frame(maxWidth: .infinity, alignment: .leading)
                    .background(C.pagoFondo, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
            }
            .padding(.vertical, 14)
            Troquel(vertical: false)
            HStack(alignment: .firstTextBaseline) {
                Text("Hoy pagás").font(Tipo.lato(16, .bold))
                Spacer()
                Text(p.displayPrice).font(Tipo.lato(32, .black)).tracking(-1).monospacedDigit()
                    .contentTransition(.numericText()).animation(Movimiento.tocar, value: p.id)
            }
            .padding(.vertical, 14)
            Button {
                Task { await comprar(p) }
            } label: {
                HStack(spacing: 8) {
                    if comprando == p.id { ProgressView().tint(C.petroleoHondo) } else { Image(systemName: "apple.logo") }
                    Text(comprando == p.id ? "Abriendo la App Store…" : "Pagar \(p.displayPrice) con Apple")
                }
            }
            .buttonStyle(BotonMenta(alto: 56, conSombra: true))
            .disabled(comprando != nil)
        }
        .foregroundStyle(C.tinta).padding(18)
        .background(C.tarjeta, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 22, style: .continuous).strokeBorder(C.linea, lineWidth: 1))
        .shadow(color: Color(hex: 0x0F2328, alfa: 0.12), radius: 20, y: 14)
    }

    private func fila(_ a: String, _ b: String) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(a).font(Tipo.lato(14)).foregroundStyle(C.sub)
            Spacer(minLength: 12)
            Text(b).font(Tipo.lato(14, .bold)).multilineTextAlignment(.trailing)
        }
    }

    private var botonRestaurar: some View {
        Button(restaurando ? "Buscando tus compras…" : "Restaurar compras") {
            Task {
                restaurando = true
                defer { restaurando = false }
                do {
                    let n = try await Compras.restaurar(estado: estado)
                    estado.avisar(n == 0 ? "No encontramos compras para restaurar" : "Listo, recuperamos tu suscripción", ok: n > 0)
                    await cargar()
                } catch { estado.avisar(error) }
            }
        }
        .buttonStyle(BotonTexto()).frame(maxWidth: .infinity).disabled(restaurando)
    }

    private var legales: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("La suscripción se cobra a tu cuenta de Apple al confirmar y se renueva sola cada mes, al mismo precio, salvo que la canceles al menos 24 horas antes del fin del período. Se administra y cancela en Ajustes › tu nombre › Suscripciones.")
                .font(Tipo.lato(12)).foregroundStyle(C.sub)
            HStack(spacing: 16) {
                Link("Términos de uso", destination: URL(string: "https://tengolibre.com.ar/terminos")!)
                Link("Privacidad", destination: URL(string: "https://tengolibre.com.ar/privacidad")!)
                Link("EULA de Apple", destination: URL(string: "https://www.apple.com/legal/internet-services/itunes/dev/stdeula/")!)
            }
            .font(Tipo.lato(12, .bold)).foregroundStyle(C.mentaTexto)
        }
        .padding(.horizontal, 4)
    }

    // MARK: Datos

    private func comprar(_ p: Product) async {
        comprando = p.id
        defer { comprando = nil }
        do {
            switch try await Compras.comprar(p, estado: estado) {
            case .listo:
                Haptico.exito()
                estado.avisar("¡Listo! Ya tenés el plan \(p.displayName)", texto: "Gracias por sumarte a Tengo Libre.")
                await cargar()
            case .pendiente:
                estado.avisar("La compra quedó pendiente", texto: "Cuando la App Store la apruebe, se activa sola.")
            case .cancelado:
                break
            }
        } catch {
            estado.avisar(error)
        }
    }

    private func cargar() async {
        do { s = try await estado.api.get("/v1/panel/suscripcion"); error = "" }
        catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
        if vigente == nil && estado.esDuena && !cargoTienda { await cargarTienda() }
    }

    private func cargarTienda() async {
        cargoTienda = false
        productos = await Compras.planes()
        cargoTienda = true
    }
}

/// Un estilo de botón u otro según el caso (ButtonStyle no es intercambiable en un ternario).
struct AnyButtonStyle: ButtonStyle {
    private let hacer: (Configuration) -> AnyView
    init<S: ButtonStyle>(_ s: S) { hacer = { AnyView(s.makeBody(configuration: $0)) } }
    func makeBody(configuration: Configuration) -> some View { hacer(configuration) }
}
