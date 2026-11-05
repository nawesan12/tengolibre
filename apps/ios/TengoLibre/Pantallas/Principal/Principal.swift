import SwiftUI

/// Las pestañas según el rol (handoff › Navegación).
enum Pestana: String, CaseIterable, Hashable {
    case hoy, agenda, espera, clientes, resenas, mas

    func titulo(profesional: Bool) -> String {
        switch self {
        case .hoy: profesional ? "Tu día" : "Hoy"
        case .agenda: "Agenda"
        case .espera: "Espera"
        case .clientes: "Clientes"
        case .resenas: "Reseñas"
        case .mas: "Más"
        }
    }

    var simbolo: String {
        switch self {
        case .hoy: "sun.max"
        case .agenda: "calendar"
        case .espera: "hourglass"
        case .clientes: "person.2"
        case .resenas: "star.bubble"
        case .mas: "line.3.horizontal"
        }
    }
}

/// Lo que se abre encima (hojas) y desde dónde.
@Observable
@MainActor
final class Navegacion {
    var pestana: Pestana = .hoy
    var turno: Turno?
    var nuevo: NuevoTurnoInicio?
    var compartir = false
    var avisos = false
    var buscar = false
    var pilaHoy = NavigationPath()
    var pilaAgenda = NavigationPath()
    var pilaEspera = NavigationPath()
    var pilaClientes = NavigationPath()
    var pilaResenas = NavigationPath()
    var pilaMas = NavigationPath()
}

/// Con qué arranca «Nuevo turno»: un hueco tocado, una clienta, una oferta que se carga a mano.
struct NuevoTurnoInicio: Identifiable, Equatable {
    var id = UUID()
    var profesionalId: String?
    var servicioId: String?
    var inicio: Date?
    var cliente: ClienteDelTurno?
    var esperaId: String?
}

struct Principal: View {
    @Environment(Estado.self) private var estado
    @State private var nav = Navegacion()
    @AppStorage("recorridoVisto") private var recorridoVisto = false

    private var pestanas: [Pestana] {
        estado.esProfesional ? [.hoy, .agenda, .resenas, .mas] : [.hoy, .agenda, .espera, .clientes, .mas]
    }

    var body: some View {
        @Bindable var nav = nav
        TabView(selection: $nav.pestana) {
            ForEach(pestanas, id: \.self) { p in
                pantalla(p)
                    .toolbar(.hidden, for: .tabBar)
                    .tag(p)
            }
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            BarraPestanas(pestanas: pestanas, elegida: $nav.pestana, profesional: estado.esProfesional)
        }
        .overlay(alignment: .bottomTrailing) {
            if [.hoy, .clientes].contains(nav.pestana) {
                BotonTurnoFlotante().padding(.trailing, 16).padding(.bottom, 49 + 16 + 34 - 20)
            }
        }
        .environment(nav)
        .sheet(item: $nav.turno) { t in
            DetalleTurno(turno: t)
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.visible)
                .presentationCornerRadius(Radio.hoja)
                .environment(nav)
        }
        .sheet(item: $nav.nuevo) { inicio in
            NuevoTurno(inicio: inicio)
                .presentationDetents([.large])
                .presentationCornerRadius(Radio.hoja)
                .environment(nav)
        }
        .sheet(isPresented: $nav.compartir) {
            NavigationStack { Compartir() }
                .presentationCornerRadius(Radio.hoja)
        }
        .sheet(isPresented: $nav.buscar) {
            Buscar().environment(nav)
        }
        .sheet(isPresented: $nav.avisos) {
            NavigationStack { Avisos() }
                .presentationCornerRadius(Radio.hoja)
                .environment(nav)
        }
        .task(id: estado.rutaPendiente) { await abrirRuta() }
        // La primera vez: el recorrido por todo lo que trae la app.
        .fullScreenCover(isPresented: Binding(get: { !recorridoVisto }, set: { if !$0 { recorridoVisto = true } })) {
            Recorrido { recorridoVisto = true }.environment(estado)
        }
    }

    @ViewBuilder
    private func pantalla(_ p: Pestana) -> some View {
        switch p {
        case .hoy: NavigationStack(path: $nav.pilaHoy) { Hoy().destinos() }
        case .agenda: NavigationStack(path: $nav.pilaAgenda) { Agenda().destinos() }
        case .espera: NavigationStack(path: $nav.pilaEspera) { ListaEspera().destinos() }
        case .clientes: NavigationStack(path: $nav.pilaClientes) { Clientes().destinos() }
        case .resenas: NavigationStack(path: $nav.pilaResenas) { Resenas().destinos() }
        case .mas: NavigationStack(path: $nav.pilaMas) { Mas().destinos() }
        }
    }

    /// Un push, un widget o un link abren su pantalla.
    private func abrirRuta() async {
        guard let r = estado.rutaPendiente else { return }
        estado.rutaPendiente = nil
        switch r {
        case .turno(let id):
            nav.pestana = .hoy
            if let t: Turno = try? await estado.api.get("/v1/panel/turnos/\(id)") { nav.turno = t }
        case .hoy: nav.pestana = .hoy
        case .espera: nav.pestana = estado.esProfesional ? .hoy : .espera
        case .avisos: nav.avisos = true
        case .resenas:
            if estado.esProfesional { nav.pestana = .resenas } else { nav.pestana = .mas; nav.pilaMas.append(DestinoMas.resenas) }
        case .nuevoTurno: nav.nuevo = NuevoTurnoInicio()
        case .suscripcion: nav.pestana = .mas; nav.pilaMas.append(DestinoMas.suscripcion)
        case .caja:
            nav.pestana = .hoy
            if estado.vePrecios && !estado.esProfesional { nav.pilaHoy.append(DestinoMas.caja) }
        case .buscar: nav.buscar = true
        case .agenda: nav.pestana = .agenda
        case .turnoCliente, .invitacion: break // los abre Raiz (valen en cualquier modo)
        }
    }
}

/// La barra de pestañas del handoff: el ítem activo con una cápsula de 56×30 detrás del ícono
/// (petróleo en claro, menta en oscuro) y una solapa menta de 40×5 pegada al borde de arriba.
struct BarraPestanas: View {
    var pestanas: [Pestana]
    @Binding var elegida: Pestana
    var profesional: Bool
    @Namespace private var ns
    @Environment(\.accessibilityReduceMotion) private var reducir

    var body: some View {
        HStack(spacing: 0) {
            ForEach(pestanas, id: \.self) { p in
                let on = p == elegida
                Button {
                    if on { return }
                    withAnimation(reducir ? nil : Movimiento.tocar) { elegida = p }
                } label: {
                    VStack(spacing: 2) {
                        ZStack {
                            if on {
                                Capsule().fill(C.seleccion).frame(width: 56, height: 30)
                                    .matchedGeometryEffect(id: "capsula", in: ns)
                            }
                            Image(systemName: on ? p.simbolo + (p == .mas ? "" : ".fill") : p.simbolo)
                                .font(.system(size: 20, weight: on ? .bold : .medium))
                                .foregroundStyle(on ? C.seleccionTexto : C.sub)
                        }
                        .frame(height: 30)
                        Text(p.titulo(profesional: profesional)).font(Tipo.lato(11, on ? .black : .bold))
                            .foregroundStyle(on ? C.tinta : C.sub)
                    }
                    .frame(maxWidth: .infinity, minHeight: 49)
                    .overlay(alignment: .top) {
                        if on {
                            UnevenRoundedRectangle(bottomLeadingRadius: 6, bottomTrailingRadius: 6).fill(C.menta)
                                .frame(width: 40, height: 5)
                                .matchedGeometryEffect(id: "solapa", in: ns)
                        }
                    }
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel(p.titulo(profesional: profesional))
                .accessibilityAddTraits(on ? .isSelected : [])
            }
        }
        .background(C.barra.background(.ultraThinMaterial).ignoresSafeArea(edges: .bottom))
        .overlay(alignment: .top) { Rectangle().fill(C.linea).frame(height: 1) }
    }
}

/// «Turno» flotando arriba de las pestañas. En sólo lectura o sin conexión, gris y con candado.
struct BotonTurnoFlotante: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav

    var body: some View {
        let lectura = estado.soloLectura, off = estado.sinConexion
        Button {
            if lectura { nav.pestana = .mas; nav.pilaMas.append(DestinoMas.suscripcion); return }
            if off { return }
            Haptico.suave()
            nav.nuevo = NuevoTurnoInicio()
        } label: {
            HStack(spacing: 8) {
                Image(systemName: lectura ? "lock.fill" : off ? "icloud.slash" : "plus").font(.system(size: 20, weight: .bold))
                Text(off ? "Sin conexión" : "Turno").font(Tipo.lato(17, .black))
            }
            .padding(.leading, 16).padding(.trailing, 22).frame(height: 56)
            .foregroundStyle(lectura || off ? C.sub : C.petroleoHondo)
            .background(lectura || off ? C.chip : C.menta, in: Capsule())
            .shadow(color: lectura || off ? .clear : Color(hex: 0x17805A, alfa: 0.75), radius: 14, y: 16)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(lectura ? "Turno, bloqueado: tu prueba terminó" : "Nuevo turno")
    }
}

/// La cabecera de petróleo: selector de negocio, título, subtítulo, acciones y lo de abajo
/// (segmentado, semana, «El día»).
struct Cabecera<Abajo: View>: View {
    var titulo: String
    var subtitulo: String = ""
    var volver: String = ""
    var conNegocio = false
    var acciones: [AccionCabecera] = []
    @ViewBuilder var abajo: Abajo
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !volver.isEmpty {
                Button { cerrar() } label: {
                    HStack(spacing: 0) {
                        Image(systemName: "chevron.left").font(.system(size: 22, weight: .semibold))
                        Text(volver).font(Tipo.lato(17))
                    }
                    .foregroundStyle(C.menta).frame(height: 44)
                }
                .padding(.vertical, -8).padding(.leading, -6)
            }
            if conNegocio, let n = estado.negocio { SelectorNegocio(actual: n) }
            HStack(alignment: .bottom, spacing: 8) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(titulo).font(Tipo.lato(34, .black)).tracking(-1).lineLimit(2).minimumScaleFactor(0.7)
                        .accessibilityAddTraits(.isHeader)
                    if !subtitulo.isEmpty { Text(subtitulo).font(Tipo.lato(15)).foregroundStyle(.white.opacity(0.78)) }
                }
                Spacer(minLength: 0)
                ForEach(acciones) { a in
                    BotonRedondo(simbolo: a.simbolo, etiqueta: a.etiqueta, sobrePetroleo: true, numero: a.numero, accion: a.accion)
                }
            }
            abajo
        }
        .foregroundStyle(.white)
        .padding(.horizontal, 16).padding(.top, 10).padding(.bottom, 36)
        .frame(maxWidth: .infinity, alignment: .leading)
        .fondoPetroleo()
    }
}

extension Cabecera where Abajo == EmptyView {
    init(titulo: String, subtitulo: String = "", volver: String = "", conNegocio: Bool = false, acciones: [AccionCabecera] = []) {
        self.init(titulo: titulo, subtitulo: subtitulo, volver: volver, conNegocio: conNegocio, acciones: acciones) { EmptyView() }
    }
}

struct AccionCabecera: Identifiable {
    var simbolo: String
    var etiqueta: String
    var numero = 0
    var accion: () -> Void
    var id: String { simbolo }
}

/// El negocio y la sucursal se eligen con un Menu arriba de Hoy y de Agenda.
struct SelectorNegocio: View {
    var actual: NegocioDeLaPersona
    @Environment(Estado.self) private var estado

    var body: some View {
        Menu {
            ForEach(estado.negocios) { n in
                Button {
                    estado.usar(n)
                    Task { await estado.refrescar() }
                } label: {
                    Label("\(n.nombre) · \(n.rolTexto)", systemImage: n.id == actual.id ? "checkmark.circle.fill" : "circle")
                }
            }
        } label: {
            HStack(spacing: 8) {
                Iniciales(nombre: actual.nombre, tam: 26, fondo: C.menta, tinta: C.petroleoHondo)
                Text(actual.nombre).font(Tipo.lato(14, .bold)).lineLimit(1)
                Image(systemName: "chevron.up.chevron.down").font(.system(size: 12, weight: .bold))
            }
            .padding(.leading, 6).padding(.trailing, 12).frame(height: 36)
            .background(.white.opacity(0.1), in: Capsule())
            .foregroundStyle(.white)
        }
        .disabled(estado.negocios.count < 2)
        .padding(.bottom, -4)
        .accessibilityLabel("Negocio: \(actual.nombre). Cambiar de negocio")
    }
}

/// El cuerpo de las pantallas con cabecera: sube 20 pt sobre la cabecera, radio 24 arriba, fondo con puntos.
struct Cuerpo<Contenido: View>: View {
    var espacio: CGFloat = 12
    @ViewBuilder var contenido: Contenido
    var body: some View {
        VStack(alignment: .leading, spacing: espacio) { contenido }
            .padding(.horizontal, 16).padding(.top, 16).padding(.bottom, 140)
            .frame(maxWidth: .infinity, alignment: .leading)
            .fondoPuntos()
            .clipShape(UnevenRoundedRectangle(topLeadingRadius: 24, topTrailingRadius: 24, style: .continuous))
            .padding(.top, -20)
    }
}
