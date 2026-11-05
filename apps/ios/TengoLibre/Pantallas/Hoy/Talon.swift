import SwiftUI

/// El talón: troquel a 76 pt del borde izquierdo, medialunas de 8 pt del color del fondo y una línea
/// punteada de 2 pt [4,4] (handoff › Firmas de la marca).
struct TalonShape: Shape {
    var troquelX: CGFloat = 76
    var radio: CGFloat = Radio.talon
    var muesca: CGFloat = 8
    func path(in r: CGRect) -> Path {
        var p = Path(roundedRect: r, cornerRadius: radio, style: .continuous)
        p = p.subtracting(Path(ellipseIn: CGRect(x: troquelX - muesca, y: -muesca, width: muesca * 2, height: muesca * 2)))
        p = p.subtracting(Path(ellipseIn: CGRect(x: troquelX - muesca, y: r.height - muesca, width: muesca * 2, height: muesca * 2)))
        return p
    }
}

/// La línea punteada del troquel.
struct Troquel: View {
    var color: Color = C.linea
    var vertical = true
    var body: some View {
        GeometryReader { g in
            Path { p in
                if vertical { p.move(to: CGPoint(x: 1, y: 0)); p.addLine(to: CGPoint(x: 1, y: g.size.height)) }
                else { p.move(to: CGPoint(x: 0, y: 1)); p.addLine(to: CGPoint(x: g.size.width, y: 1)) }
            }
            .stroke(color, style: StrokeStyle(lineWidth: 2, dash: [4, 4]))
        }
        .frame(width: vertical ? 2 : nil, height: vertical ? nil : 2)
        .accessibilityHidden(true)
    }
}

/// Un turno como talón. A la izquierda la hora (900, 19 pt) y la duración; a la derecha el nombre,
/// el servicio y la pastilla de estado. En tamaños de accesibilidad, la hora va arriba.
struct VistaTalon: View {
    var turno: Turno
    var mostrarProfesional = true
    var ahora = Date()
    var alTocar: () -> Void = {}
    var alDeslizar: ((String) -> Void)? = nil
    @Environment(\.dynamicTypeSize) private var letra
    @State private var corrimiento: CGFloat = 0
    @State private var paso = false

    private var est: EstadoTurno { turno.visible(ahora) }
    private var nuevo: Bool { turno.esNuevo(ahora) }
    private var tinta: Color { nuevo ? C.petroleoHondo : C.tinta }
    private var sub: Color { nuevo ? Color(hex: 0x1D4A3A) : C.sub }

    var body: some View {
        ZStack(alignment: .trailing) {
            if alDeslizar != nil, corrimiento < 0 { acciones }
            contenido
                .offset(x: corrimiento)
                .gesture(alDeslizar == nil ? nil : arrastre)
        }
        .contextMenu { menu }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(Formato.horaDe(turno.inicio)), \(turno.cliente.nombre), \(turno.servicio). \(est.texto)")
        .accessibilityAction(named: "Ver turno", alTocar)
        .accessibilityActions {
            if let alDeslizar {
                Button("Marcar atendido") { alDeslizar("atendido") }
                Button("Marcar ausente") { alDeslizar("ausente") }
            }
        }
    }

    private var contenido: some View {
        Button(action: alTocar) {
            ViewThatFits(in: .horizontal) {
                if !letra.isAccessibilitySize { horizontal }
                vertical
            }
            .background(nuevo ? C.menta : C.tarjeta, in: TalonShape(troquelX: letra.isAccessibilitySize ? -20 : 76))
            .overlay {
                if est == .enCurso {
                    TalonShape(troquelX: letra.isAccessibilitySize ? -20 : 76).stroke(C.seleccion, lineWidth: 2)
                }
            }
            .shadow(color: Color(hex: 0x0F2328, alfa: 0.18), radius: 14, y: 10)
            .opacity(est == .atendido || est == .cancelado ? 0.55 : 1)
        }
        .buttonStyle(.plain)
    }

    private var horizontal: some View {
        HStack(spacing: 0) {
            VStack(spacing: 2) {
                Text(Formato.horaDe(turno.inicio)).font(Tipo.lato(19, .black)).tracking(-0.4).monospacedDigit()
                Text(Formato.duracion(turno.minutos)).font(Tipo.lato(12)).foregroundStyle(sub).monospacedDigit()
            }
            .frame(width: 76)
            Troquel(color: nuevo ? Color(hex: 0x102B31, alfa: 0.35) : C.linea)
            datos.padding(.horizontal, 14).padding(.vertical, 10)
        }
        .foregroundStyle(tinta)
        .frame(minHeight: 80)
    }

    private var vertical: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(Formato.horaDe(turno.inicio)).font(Tipo.lato(19, .black)).monospacedDigit()
                Text(Formato.duracion(turno.minutos)).font(Tipo.lato(12)).foregroundStyle(sub)
            }
            Troquel(color: C.linea, vertical: false)
            datos
        }
        .foregroundStyle(tinta)
        .padding(14)
    }

    private var datos: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(turno.cliente.nombre).font(Tipo.lato(17, .bold)).lineLimit(1)
            Text(mostrarProfesional && !turno.profesional.isEmpty ? "\(turno.servicio) · \(turno.profesional)" : turno.servicio)
                .font(Tipo.lato(13)).foregroundStyle(sub).lineLimit(1)
            if nuevo {
                PastillaEstado(texto: "Nueva · recién", simbolo: "bolt.fill", fondo: C.petroleoHondo, tinta: C.menta)
            } else {
                PastillaEstado(texto: est.texto, simbolo: est.simbolo, fondo: est.fondo, tinta: est.tinta)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    // MARK: Deslizar: sigue el dedo; pasado el umbral, háptico de selección y quedan las dos acciones.

    private var acciones: some View {
        HStack(spacing: 6) {
            Button { cerrarYHacer("atendido") } label: {
                VStack(spacing: 4) { Image(systemName: "checkmark.circle").font(.system(size: 22, weight: .bold)); Text("Atendido").font(Tipo.lato(13, .black)) }
                    .frame(width: 76).frame(maxHeight: .infinity)
                    .foregroundStyle(C.petroleoHondo).background(C.menta, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            }
            Button { cerrarYHacer("ausente") } label: {
                VStack(spacing: 4) { Image(systemName: "person.slash").font(.system(size: 22, weight: .bold)); Text("Ausente").font(Tipo.lato(13, .bold)) }
                    .frame(width: 76).frame(maxHeight: .infinity)
                    .foregroundStyle(.white).background(C.petroleo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            }
        }
        .buttonStyle(.plain)
        .accessibilityHidden(true)
    }

    private var arrastre: some Gesture {
        DragGesture(minimumDistance: 18)
            .onChanged { v in
                guard abs(v.translation.width) > abs(v.translation.height) else { return }
                corrimiento = min(0, max(-190, v.translation.width + (paso ? -164 : 0)))
                if corrimiento < -100, !paso { paso = true; Haptico.seleccion() }
                if corrimiento > -100, paso { paso = false }
            }
            .onEnded { _ in
                withAnimation(Movimiento.entrar) { corrimiento = paso ? -164 : 0 }
            }
    }

    private func cerrarYHacer(_ estado: String) {
        withAnimation(Movimiento.entrar) { corrimiento = 0; paso = false }
        alDeslizar?(estado)
    }

    @ViewBuilder private var menu: some View {
        Button("Ver turno", systemImage: "doc.text", action: alTocar)
        if let alDeslizar {
            if EstadoTurno.puedePasar(turno.estado, "confirmado"), turno.estado != "confirmado" {
                Button("Confirmar", systemImage: "checkmark") { alDeslizar("confirmado") }
            }
            if EstadoTurno.puedePasar(turno.estado, "atendido") { Button("Atendido", systemImage: "checkmark.circle") { alDeslizar("atendido") } }
            if EstadoTurno.puedePasar(turno.estado, "ausente") { Button("No vino", systemImage: "person.slash") { alDeslizar("ausente") } }
            if EstadoTurno.puedePasar(turno.estado, "cancelado") { Button("Cancelar", systemImage: "calendar.badge.minus", role: .destructive) { alDeslizar("cancelado") } }
        }
        if !turno.cliente.telefono.isEmpty, let u = URL(string: "tel:\(turno.cliente.telefono.filter { "+0123456789".contains($0) })") {
            Link(destination: u) { Label("Llamar", systemImage: "phone") }
        }
    }
}

/// Un hueco (tiempo libre): rayas que corren, «LIBRE · 45 MIN», la hora con quién, y la acción.
struct VistaHueco: View {
    var hueco: Dia.Hueco
    var accion: String
    var simboloAccion: String = "megaphone"
    var avisado = false
    var alTocar: () -> Void
    var alAccion: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            VStack(alignment: .leading, spacing: 2) {
                PastillaPapel { Text("LIBRE · \(Formato.duracion(hueco.minutos).uppercased())").font(Tipo.lato(12, .black)).tracking(0.96) }
                    .padding(.leading, -8)
                PastillaPapel(radio: 8) { Text("\(Dia.hhmm(hueco.desde)) · \(hueco.profesional)").font(Tipo.lato(17, .bold)).monospacedDigit() }
                    .padding(.leading, -8)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            Button(action: alAccion) {
                HStack(spacing: 6) {
                    Image(systemName: avisado ? "checkmark" : simboloAccion).font(.system(size: 16, weight: .bold))
                    Text(avisado ? "Avisado" : accion).font(Tipo.lato(14, .black))
                }
                .padding(.leading, 10).padding(.trailing, 14).frame(height: 44)
                .foregroundStyle(C.petroleoHondo)
                .background(C.menta, in: Capsule())
                .shadow(color: Color(hex: 0x17805A, alfa: 0.9), radius: 8, y: 8)
            }
            .buttonStyle(.plain)
            .disabled(avisado)
        }
        .padding(.leading, 16).padding([.vertical, .trailing], 10)
        .frame(minHeight: 72)
        .hueco(Radio.talon, quieto: avisado)
        .contentShape(Rectangle())
        .onTapGesture(perform: alTocar)
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Libre \(Formato.duracion(hueco.minutos)), \(Dia.hhmm(hueco.desde)) con \(hueco.profesional)")
    }
}

/// «El día» en la cabecera: una fila por profesional, de la primera a la última hora, con lo ocupado
/// en crema al 88 %, lo libre rayado y la línea menta de ahora. Para VoiceOver, un solo elemento.
struct ElDia: View {
    var dias: [Dia.DeProfesional]
    var esHoy: Bool
    @Environment(\.accessibilityReduceMotion) private var reducir

    var body: some View {
        let (desde, hasta) = Dia.rango(dias)
        let total = CGFloat(max(60, hasta - desde))
        let libres = dias.reduce(0) { $0 + $1.libres }
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Text("EL DÍA").font(Tipo.lato(12, .black)).tracking(0.96).foregroundStyle(.white.opacity(0.75))
                Spacer()
                HStack(spacing: 4) {
                    Rayas(quieto: true, paso: 5, color: C.menta, fondo: .clear).frame(width: 14, height: 10).clipShape(RoundedRectangle(cornerRadius: 3))
                    Text(Formato.libres(libres)).font(Tipo.lato(12, .black)).foregroundStyle(.white.opacity(0.75))
                }
            }
            ZStack(alignment: .topLeading) {
                VStack(spacing: 6) {
                    ForEach(dias) { d in
                        HStack(spacing: 8) {
                            Text(d.nombre.split(separator: " ").first.map(String.init) ?? d.nombre)
                                .font(Tipo.lato(13, .bold)).foregroundStyle(.white.opacity(0.85))
                                .frame(width: 44, alignment: .leading).lineLimit(1).minimumScaleFactor(0.7)
                            GeometryReader { g in
                                ZStack(alignment: .leading) {
                                    RoundedRectangle(cornerRadius: 4).fill(.white.opacity(0.06))
                                    ForEach(d.franjas, id: \.self) { f in
                                        Rayas(quieto: true, paso: 6, color: C.menta, fondo: C.menta.opacity(0.12))
                                            .clipShape(RoundedRectangle(cornerRadius: 4))
                                            .frame(width: g.size.width * CGFloat(f.largo) / total)
                                            .offset(x: g.size.width * CGFloat(f.desde - desde) / total)
                                    }
                                    ForEach(d.ocupado.indices, id: \.self) { i in
                                        let o = d.ocupado[i]
                                        RoundedRectangle(cornerRadius: 4).fill(C.crema.opacity(0.88))
                                            .frame(width: max(2, g.size.width * CGFloat(o.tramo.largo) / total - 2))
                                            .offset(x: g.size.width * CGFloat(o.tramo.desde - desde) / total)
                                    }
                                }
                            }
                            .frame(height: 16)
                        }
                    }
                }
                if esHoy {
                    let m = Dia.minutosAhora()
                    if m >= desde && m <= hasta {
                        GeometryReader { g in
                            Rectangle().fill(C.menta).frame(width: 2)
                                .offset(x: 52 + (g.size.width - 52) * CGFloat(m - desde) / total - 1)
                        }
                        .padding(.vertical, -5)
                    }
                }
            }
            HStack {
                ForEach(marcas(desde, hasta), id: \.self) { h in
                    Text("\(h / 60)").font(Tipo.lato(11, .bold)).monospacedDigit().foregroundStyle(.white.opacity(0.6))
                    if h != marcas(desde, hasta).last { Spacer() }
                }
            }
            .padding(.leading, 52)
        }
        .padding(.horizontal, 14).padding(.top, 14).padding(.bottom, 12)
        .background(.white.opacity(0.07), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
        .overlay(alignment: .top) { RoundedRectangle(cornerRadius: 20).stroke(.white.opacity(0.1), lineWidth: 1).mask(Rectangle().frame(height: 1)) }
        .accessibilityElement()
        .accessibilityLabel(resumenHablado)
    }

    private func marcas(_ desde: Int, _ hasta: Int) -> [Int] {
        let paso = max(60, ((hasta - desde) / 4 / 60) * 60)
        var out: [Int] = []
        var h = desde
        while h < hasta { out.append(h); h += paso }
        out.append(hasta)
        return out
    }

    private var resumenHablado: String {
        "El día: " + dias.map { d in
            let ocup = d.ocupado.count
            return "\(d.nombre), \(ocup) \(ocup == 1 ? "turno" : "turnos"), \(Formato.libres(d.libres))"
        }.joined(separator: "; ")
    }
}
