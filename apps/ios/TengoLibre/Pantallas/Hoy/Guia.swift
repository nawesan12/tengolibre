import SwiftUI

// La app guía sola (Hoy nunca queda sin caminos): atajos a todo lo del negocio, «Descubrí» con lo que
// todavía no se usa y el recorrido de la primera vez.

/// Los atajos de Hoy: lo que vive en «Más», a un toque y sin salir de Hoy.
struct AtajosHoy: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav

    private struct Atajo: Identifiable { var simbolo: String; var titulo: String; var destino: DestinoMas?; var accion: (() -> Void)?; var id: String { titulo } }

    private var atajos: [Atajo] {
        let g = estado.gestiona, mostrador = g || estado.rol == "recepcion"
        if estado.esProfesional {
            return [Atajo(simbolo: "clock", titulo: "Mi horario", destino: .miHorario, accion: nil),
                    Atajo(simbolo: "beach.umbrella", titulo: "Días libres", destino: .miHorario, accion: nil),
                    Atajo(simbolo: "star.bubble", titulo: "Reseñas", destino: nil) { nav.pestana = .resenas },
                    Atajo(simbolo: "calendar.badge.plus", titulo: "En mi calendario", destino: .integraciones, accion: nil)]
        }
        var a: [Atajo] = [Atajo(simbolo: "banknote", titulo: "Caja de hoy", destino: .caja, accion: nil)]
        if g { a.append(Atajo(simbolo: "creditcard", titulo: "Señas", destino: .cobros, accion: nil)) }
        if mostrador { a.append(Atajo(simbolo: "ticket", titulo: "Membresías", destino: .membresias, accion: nil)) }
        if g { a.append(Atajo(simbolo: "chart.bar", titulo: "Reportes", destino: .reportes, accion: nil)) }
        if mostrador { a.append(Atajo(simbolo: "tag", titulo: "Cupones", destino: .cupones, accion: nil)) }
        if g {
            a.append(Atajo(simbolo: "sparkles", titulo: "IA", destino: .integraciones, accion: nil))
            a.append(Atajo(simbolo: "globe", titulo: "Mi web", destino: .miWeb, accion: nil))
            a.append(Atajo(simbolo: "person.2.badge.gearshape", titulo: "Equipo", destino: .equipo, accion: nil))
        }
        a.append(Atajo(simbolo: "square.and.arrow.up", titulo: "Mi link", destino: nil) { nav.compartir = true })
        return a
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("TU NEGOCIO").rotulo()
                Spacer()
                Button("Ver todo") { nav.pestana = .mas }.font(Tipo.lato(13, .bold)).foregroundStyle(C.mentaTexto)
            }
            .padding(.horizontal, 4)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(atajos) { a in
                        Button {
                            Haptico.suave()
                            if let d = a.destino { nav.pilaHoy.append(d) } else { a.accion?() }
                        } label: {
                            VStack(alignment: .leading, spacing: 8) {
                                Image(systemName: a.simbolo).font(.system(size: 20, weight: .semibold)).foregroundStyle(C.mentaTexto)
                                    .frame(width: 36, height: 36).background(C.mentaSuave, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                                Text(a.titulo).font(Tipo.lato(14, .bold)).foregroundStyle(C.tinta).lineLimit(2).multilineTextAlignment(.leading)
                            }
                            .padding(12).frame(width: 104, height: 96, alignment: .topLeading)
                            .tarjeta(18)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 1).padding(.vertical, 2)
            }
            .padding(.horizontal, -16).contentMargins(.horizontal, 16, for: .scrollContent)
        }
    }
}

/// «Descubrí»: una tarjeta por vez con algo que el negocio todavía no aprovecha, con su botón directo.
struct DescubrirHoy: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var ideas: [Idea] = []
    @AppStorage("descubrirDescartadas") private var descartadas = ""

    struct Idea: Identifiable, Equatable {
        var id: String
        var simbolo: String
        var titulo: String
        var texto: String
        var accion: String
        var destino: DestinoMas?
    }

    private var visibles: [Idea] { ideas.filter { !descartadas.split(separator: ",").map(String.init).contains($0.id) } }

    var body: some View {
        Group {
            if let i = visibles.first {
                VStack(alignment: .leading, spacing: 12) {
                    HStack(alignment: .top, spacing: 12) {
                        Image(systemName: i.simbolo).font(.system(size: 22, weight: .semibold)).foregroundStyle(C.menta)
                            .frame(width: 44, height: 44).background(.white.opacity(0.1), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                        VStack(alignment: .leading, spacing: 4) {
                            Text("DESCUBRÍ · \(ideas.count - visibles.count + 1) DE \(ideas.count)").font(Tipo.lato(11, .black)).tracking(0.8).foregroundStyle(C.menta)
                            Text(i.titulo).font(Tipo.lato(17, .bold))
                            Text(i.texto).font(Tipo.lato(14)).foregroundStyle(.white.opacity(0.8))
                        }
                    }
                    HStack(spacing: 8) {
                        Button(i.accion) { abrir(i) }.buttonStyle(BotonMenta(alto: 44, tam: 15, conSombra: false))
                        Button("Ahora no") { withAnimation(Movimiento.entrar) { descartar(i) } }
                            .font(Tipo.lato(15, .bold)).foregroundStyle(.white.opacity(0.85)).frame(height: 44).padding(.horizontal, 12)
                    }
                }
                .foregroundStyle(.white).padding(16).frame(maxWidth: .infinity, alignment: .leading)
                .background { ZStack { C.bloquePetroleo; Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5) } }
                .clipShape(RoundedRectangle(cornerRadius: 22, style: .continuous))
                .transition(.asymmetric(insertion: .move(edge: .trailing).combined(with: .opacity), removal: .move(edge: .leading).combined(with: .opacity)))
                .id(i.id)
            }
        }
        .task(id: estado.version) { await cargar() }
    }

    private func abrir(_ i: Idea) {
        Haptico.suave()
        if i.id == "widget" {
            descartar(i)
            nav.pilaHoy.append(DestinoMas.recorrido)
        } else if let d = i.destino {
            nav.pilaHoy.append(d)
        } else if i.id == "compartir" {
            nav.compartir = true
        }
    }

    private func descartar(_ i: Idea) { descartadas = (descartadas.split(separator: ",").map(String.init) + [i.id]).joined(separator: ",") }

    private struct MP: Decodable { var disponible: Bool; var conectada: Bool }
    private struct IA: Decodable { var activo: Bool }
    private struct Planes: Decodable { var planes: [PlanMembresia] }
    private struct Recs: Decodable { struct R: Decodable { var activo: Bool }; var recordatorios: [R] }
    private struct Cups: Decodable { var cupones: [Cupon] }
    private struct Fotos: Decodable { var fotos: [FotoGaleria] }

    private func cargar() async {
        var out: [Idea] = []
        if estado.esProfesional {
            out.append(Idea(id: "calendario", simbolo: "calendar.badge.plus", titulo: "Tus turnos en tu calendario", texto: "Suscribite desde Google o Apple y te aparecen solos.", accion: "Conectar", destino: .integraciones))
            out.append(Idea(id: "widget", simbolo: "apps.iphone", titulo: "Tu próximo turno, sin abrir la app", texto: "Sumá el widget de Tengo Libre a tu pantalla de inicio.", accion: "Cómo se hace", destino: nil))
            ideas = out
            return
        }
        if estado.gestiona {
            if let m: MP = try? await estado.api.get("/v1/panel/mercadopago"), m.disponible, !m.conectada {
                out.append(Idea(id: "mp", simbolo: "creditcard", titulo: "Cobrá señas al reservar", texto: "Con Mercado Pago nadie reserva y no viene. La plata va directo a tu cuenta.", accion: "Conectar", destino: .cobros))
            }
            if let i: IA = try? await estado.api.get("/v1/panel/ia/publico"), !i.activo {
                out.append(Idea(id: "ia", simbolo: "sparkles", titulo: "Que te reserven desde ChatGPT y Claude", texto: "Tus horarios libres quedan a mano de los asistentes de IA.", accion: "Activar", destino: .integraciones))
            }
            out.append(Idea(id: "mcp", simbolo: "bubble.left.and.text.bubble.right", titulo: "Manejá tu agenda hablando", texto: "Conectá Claude o ChatGPT y pediles «¿qué tengo mañana?» o «mové a Sofía al jueves».", accion: "Ver cómo", destino: .integraciones))
            if let f: Fotos = try? await estado.api.get("/v1/panel/mi-web/galeria"), f.fotos.isEmpty {
                out.append(Idea(id: "fotos", simbolo: "photo.on.rectangle", titulo: "Sumá fotos a tu página", texto: "Las páginas con fotos reciben más reservas.", accion: "Subir fotos", destino: .miWeb))
            }
        }
        if estado.gestiona || estado.rol == "recepcion" {
            if let p: Planes = try? await estado.api.get("/v1/panel/membresias/planes"), p.planes.isEmpty {
                out.append(Idea(id: "membresias", simbolo: "ticket", titulo: "Vendé membresías", texto: "Un abono mensual con visitas incluidas: plata fija todos los meses.", accion: "Crear un plan", destino: .membresias))
            }
            if let r: Recs = try? await estado.api.get("/v1/panel/recordatorios"), !r.recordatorios.contains(where: \.activo) {
                out.append(Idea(id: "recordatorios", simbolo: "envelope.badge", titulo: "Que vuelvan solas", texto: "Un mail en su cumpleaños y otro si hace mucho que no vienen.", accion: "Prender", destino: .avisosClientes))
            }
            if let c: Cups = try? await estado.api.get("/v1/panel/cupones"), c.cupones.isEmpty {
                out.append(Idea(id: "cupones", simbolo: "tag", titulo: "Llená las horas flojas", texto: "Un cupón para los días que siempre quedan vacíos.", accion: "Ver mis horas muertas", destino: .reportes))
            }
        }
        out.append(Idea(id: "caja", simbolo: "banknote", titulo: "Tu caja, en el bolsillo", texto: "Cobrás cada turno desde la app y ves cuánto entró hoy, por medio de pago.", accion: "Ver la caja", destino: .caja))
        out.append(Idea(id: "widget", simbolo: "apps.iphone", titulo: "Tu día en la pantalla de inicio", texto: "El widget te muestra el próximo turno sin abrir la app.", accion: "Cómo se hace", destino: nil))
        out.append(Idea(id: "compartir", simbolo: "qrcode", titulo: "Que te reserven solos", texto: "Poné tu link en Instagram y WhatsApp, o imprimí el QR para el mostrador.", accion: "Compartir mi link", destino: nil))
        withAnimation(Movimiento.entrar) { ideas = out }
    }
}

/// El recorrido de la primera vez (y desde Más › «Qué podés hacer»): pocas pantallas, poco texto.
struct Recorrido: View {
    var alTerminar: () -> Void
    @Environment(Estado.self) private var estado
    @State private var pagina = 0

    private struct Paso: Hashable { var simbolo: String; var titulo: String; var texto: String; var puntos: [String] }

    private var pasos: [Paso] {
        if estado.esProfesional {
            return [
                Paso(simbolo: "sun.max", titulo: "Tu día, de un vistazo", texto: "Tus turnos de hoy en orden, con el que está pasando arriba.", puntos: ["Deslizá un turno para marcarlo atendido", "Tocalo para ver la clienta y llamarla"]),
                Paso(simbolo: "beach.umbrella", titulo: "Tu horario y tus días libres", texto: "Desde Más los pedís y te avisamos cuando los aprueban.", puntos: ["Bloqueá un rato o un día", "Pedí vacaciones sin mandar mensajes"]),
                Paso(simbolo: "bell.badge", titulo: "Enterate al toque", texto: "Cada reserva o cancelación te llega como aviso.", puntos: ["Sumá el widget a tu pantalla de inicio", "La Isla Dinámica te muestra el turno en curso"])
            ]
        }
        return [
            Paso(simbolo: "sun.max", titulo: "Hoy, siempre al día", texto: "Los turnos de hoy, los huecos libres y lo que hay que atender. Se actualiza solo.", puntos: ["Deslizá un turno: atendido o ausente", "Tocá un hueco para cargar un turno", "«Avisar a 3» ofrece el hueco a la lista de espera"]),
            Paso(simbolo: "banknote", titulo: "La plata del negocio", texto: "Cobrás cada turno desde la app y ves la caja del día.", puntos: ["Señas con Mercado Pago al reservar", "Membresías y cupones", "Reportes: ingresos, ocupación y horas muertas"]),
            Paso(simbolo: "person.2", titulo: "Tus clientas y tu equipo", texto: "Fichas con historial, notas y etiquetas. Horarios y días libres de cada profesional.", puntos: ["Penalidades para quien falta sin avisar", "Recordatorios de cumpleaños y «te extrañamos»"]),
            Paso(simbolo: "sparkles", titulo: "Tu web y la IA", texto: "Tu página de reservas se edita desde acá, y la IA trabaja para vos.", puntos: ["Que te reserven desde ChatGPT y Claude", "Manejá la agenda hablándole a tu asistente", "Logo, fotos, colores y plantilla"]),
            Paso(simbolo: "bell.badge", titulo: "Todo te avisa", texto: "Reservas, cancelaciones y huecos que se liberan, con botones para resolver sin abrir la app.", puntos: ["Widget con tu próximo turno", "La Isla Dinámica con el turno en curso", "Todo lo demás, en la pestaña Más"])
        ]
    }

    var body: some View {
        VStack(spacing: 0) {
            TabView(selection: $pagina) {
                ForEach(Array(pasos.enumerated()), id: \.offset) { i, p in
                    VStack(alignment: .leading, spacing: 18) {
                        Spacer()
                        Image(systemName: p.simbolo).font(.system(size: 34, weight: .semibold)).foregroundStyle(C.petroleoHondo)
                            .frame(width: 76, height: 76).background(C.menta, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
                        Text(p.titulo).font(Tipo.lato(32, .black)).tracking(-0.8).foregroundStyle(.white)
                        Text(p.texto).font(Tipo.lato(17)).foregroundStyle(.white.opacity(0.82))
                        VStack(alignment: .leading, spacing: 10) {
                            ForEach(p.puntos, id: \.self) { x in
                                Label { Text(x).font(Tipo.lato(15, .bold)) } icon: { Image(systemName: "checkmark.circle.fill").foregroundStyle(C.menta) }
                                    .foregroundStyle(.white)
                            }
                        }
                        .padding(.top, 4)
                        Spacer()
                        Spacer()
                    }
                    .padding(.horizontal, 28)
                    .tag(i)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            HStack(spacing: 6) {
                ForEach(0..<pasos.count, id: \.self) { i in
                    Capsule().fill(i == pagina ? C.menta : .white.opacity(0.3)).frame(width: i == pagina ? 20 : 6, height: 6)
                }
            }
            .animation(Movimiento.tocar, value: pagina)
            .padding(.bottom, 20)
            VStack(spacing: 8) {
                Button(pagina == pasos.count - 1 ? "Empezar" : "Siguiente") {
                    if pagina == pasos.count - 1 { alTerminar() } else { withAnimation(Movimiento.entrar) { pagina += 1 } }
                }
                .buttonStyle(BotonMenta())
                Button("Saltear") { alTerminar() }.font(Tipo.lato(15, .bold)).foregroundStyle(.white.opacity(0.75)).frame(height: 44)
                    .opacity(pagina == pasos.count - 1 ? 0 : 1)
            }
            .padding(.horizontal, 20).padding(.bottom, 12)
        }
        .background { ZStack { C.petroleo; Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5) }.ignoresSafeArea() }
    }
}

/// Más › «Qué podés hacer»: el recorrido otra vez y cómo sumar el widget.
struct RecorridoOtraVez: View {
    @Environment(\.dismiss) private var cerrar
    @State private var tour = false

    var body: some View {
        PantallaMas(titulo: "Qué podés hacer", subtitulo: "Todo lo que trae Tengo Libre", recargar: {}) {
            Button { tour = true } label: { FilaLista(simbolo: "play.circle", titulo: "Ver el recorrido", detalle: "Un minuto, cinco pantallas") }
                .buttonStyle(.plain).tarjeta()
            Bloque(titulo: "Sumar el widget", pie: "Muestra tu próximo turno y lo que sigue. También está para la pantalla bloqueada.") {
                Renglon(titulo: "1. Mantené apretada la pantalla de inicio", simbolo: "hand.tap")
                Renglon(titulo: "2. Tocá «Editar» › «Agregar widget»", simbolo: "plus.app")
                Renglon(titulo: "3. Buscá «Tengo Libre» y elegí el tamaño", simbolo: "magnifyingglass")
            }
            Bloque(titulo: "La Isla Dinámica", pie: "Cuando empieza un turno con la app abierta, queda arriba con el tiempo que falta y el botón «Atendido».") {
                Renglon(titulo: "Turno en curso, siempre a la vista", simbolo: "timelapse")
            }
            Bloque(titulo: "Avisos con botones", pie: "Mantené apretado un aviso para confirmar, avisar a la lista o responder una reseña sin abrir la app.") {
                Renglon(titulo: "Resolvé desde la notificación", simbolo: "bell.badge")
            }
        }
        .fullScreenCover(isPresented: $tour) { Recorrido { tour = false } }
    }
}
