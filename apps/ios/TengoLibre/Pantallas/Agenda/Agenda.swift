import SwiftUI

/// 07 Agenda · día y 08 Agenda · semana.
struct Agenda: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var datos = DatosDia()
    @State private var modo = "dia"
    @State private var dia = Formato.calendario.startOfDay(for: Date())
    @State private var indice = 1 // 0 = Todos
    @State private var ahora = Date()

    private var lunes: Date {
        let cal = Formato.calendario
        return cal.date(from: cal.dateComponents([.yearForWeekOfYear, .weekOfYear], from: dia)) ?? dia
    }
    private var semana: [Date] { (0..<7).compactMap { Formato.calendario.date(byAdding: .day, value: $0, to: lunes) } }
    private var pros: [Profesional] {
        Dia.profesionalesDelDia(datos.profesionales, datos.turnos, rol: estado.rol)
    }
    private var esHoy: Bool { Formato.diaISO(dia) == Formato.diaISO(Date()) }
    private var dias: [Dia.DeProfesional] {
        Dia.armar(pros, datos.turnos, fecha: dia, ahora: esHoy ? Dia.minutosAhora(ahora) : nil)
    }
    private var elegida: Dia.DeProfesional? { indice == 0 || dias.isEmpty ? nil : dias[min(indice - 1, dias.count - 1)] }

    var body: some View {
        ZStack(alignment: .bottom) {
            ScrollView {
                VStack(spacing: 0) {
                    Cabecera(titulo: "Agenda",
                             subtitulo: modo == "dia" ? Formato.fechaLarga(dia) : "semana del \(Formato.calendario.component(.day, from: semana[0])) al \(Formato.dia(semana[6]))",
                             conNegocio: modo == "dia" && !estado.esProfesional,
                             acciones: modo == "dia" ? [
                                AccionCabecera(simbolo: "square.and.arrow.up", etiqueta: "Compartir tu link") { nav.compartir = true },
                                AccionCabecera(simbolo: "bell", etiqueta: "Avisos", numero: estado.sinLeer) { nav.avisos = true }
                             ] : []) {
                        Segmentado(opciones: [("dia", "Día"), ("semana", "Semana")], elegido: $modo)
                        if modo == "dia" { tiraSemana }
                    }
                    Cuerpo {
                        if datos.carga == .cargando && datos.turnos.isEmpty {
                            ForEach(0..<5, id: \.self) { _ in Esqueleto(alto: 70) }
                        } else if case .error(let m) = datos.carga {
                            ErrorConSalida(titulo: "No pudimos traer la agenda", texto: m) { Task { await cargar() } }
                        } else if modo == "dia" {
                            if let p = elegida { lineaDeTiempo(p) } else { todos }
                        } else {
                            vistaSemana
                        }
                    }
                }
            }
            .scrollIndicators(.hidden)
            .gesture(DragGesture(minimumDistance: 40).onEnded { v in
                guard modo == "dia", abs(v.translation.width) > abs(v.translation.height) * 1.5 else { return }
                withAnimation(Movimiento.entrar) { indice = (indice + (v.translation.width < 0 ? 1 : -1) + dias.count + 1) % (dias.count + 1) }
                Haptico.seleccion()
            })
            if modo == "dia" && dias.count > 1 { capsula.padding(.bottom, 16) }
        }
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await cargar() }
        .task(id: "\(estado.version)-\(Formato.diaISO(lunes))-\(estado.negocio?.id ?? "")") { await cargar() }
    }

    private func cargar() async {
        await datos.cargar(estado, desde: semana[0], hasta: semana[6])
        if estado.esProfesional { indice = 1 }
    }

    // MARK: La semana arriba, con el anillo menta en el día elegido.

    private var tiraSemana: some View {
        HStack {
            ForEach(semana, id: \.self) { d in
                let on = Formato.diaISO(d) == Formato.diaISO(dia)
                let hay = datos.turnos.contains { Formato.diaISO($0.inicio) == Formato.diaISO(d) && $0.ocupa }
                Button { withAnimation(Movimiento.tocar) { dia = d }; Haptico.seleccion() } label: {
                    VStack(spacing: 2) {
                        Text(String(Formato.fechaLarga(d).prefix(1)).uppercased()).font(Tipo.lato(12, .bold)).foregroundStyle(.white.opacity(0.7))
                        Text("\(Formato.calendario.component(.day, from: d))").font(Tipo.lato(18, .bold)).monospacedDigit()
                        Circle().fill(C.menta).frame(width: 5, height: 5).opacity(hay ? 1 : 0)
                    }
                    .frame(width: 44, height: 60)
                    .overlay(Capsule().strokeBorder(on ? C.menta : .clear, lineWidth: 2))
                }
                .buttonStyle(.plain)
                .accessibilityLabel(Formato.fechaLarga(d) + (hay ? ", con turnos" : ""))
                .accessibilityAddTraits(on ? .isSelected : [])
                if d != semana.last { Spacer(minLength: 0) }
            }
        }
        .accessibilityAction(named: "Semana anterior") { cambiarSemana(-7) }
        .accessibilityAction(named: "Semana siguiente") { cambiarSemana(7) }
        .gesture(DragGesture(minimumDistance: 30).onEnded { v in cambiarSemana(v.translation.width < 0 ? 7 : -7) })
    }

    private func cambiarSemana(_ dias: Int) {
        if let d = Formato.calendario.date(byAdding: .day, value: dias, to: dia) { withAnimation { dia = d } }
    }

    // MARK: Una profesional: la línea de tiempo (76 pt por hora).

    private let porHora: CGFloat = 76

    private func lineaDeTiempo(_ p: Dia.DeProfesional) -> some View {
        let (desde, hasta) = Dia.rango([p])
        let y = { (m: Int) -> CGFloat in CGFloat(m - desde) / 60 * porHora }
        let lectura = estado.soloLectura || estado.sinConexion
        return ZStack(alignment: .topLeading) {
            ForEach(Array(stride(from: desde, through: hasta, by: 60)), id: \.self) { h in
                HStack(alignment: .top, spacing: 8) {
                    Text("\(h / 60):00").font(Tipo.lato(12, .bold)).foregroundStyle(C.sub).monospacedDigit().frame(width: 40, alignment: .leading).offset(y: -8)
                    Rectangle().fill(C.linea).frame(height: 1)
                }
                .offset(y: y(h))
            }
            ForEach(p.ocupado.indices, id: \.self) { i in
                let o = p.ocupado[i]
                bloqueTalon(o.turno).frame(height: max(30, y(o.tramo.hasta) - y(o.tramo.desde)) - 4)
                    .padding(.leading, 48).offset(y: y(o.tramo.desde) + 2)
            }
            ForEach(p.huecos) { h in
                Button { if !lectura { nav.nuevo = NuevoTurnoInicio(profesionalId: h.profesionalId, inicio: Dia.fecha(dia, minutos: h.desde)) } } label: {
                    HStack(spacing: 8) {
                        PastillaPapel {
                            Label("Libre \(Formato.duracion(h.minutos)) · tocá para cargar", systemImage: "plus.circle").font(Tipo.lato(14, .black))
                        }
                        Spacer()
                    }
                    .padding(.horizontal, 12).frame(maxWidth: .infinity, maxHeight: .infinity)
                    .hueco(14)
                }
                .buttonStyle(.plain)
                .frame(height: max(30, y(h.hasta) - y(h.desde)) - 4)
                .padding(.leading, 48).offset(y: y(h.desde) + 2)
            }
            if esHoy {
                let m = Dia.minutosAhora(ahora)
                if m >= desde && m <= hasta {
                    HStack(spacing: 0) {
                        Circle().fill(C.menta).frame(width: 10, height: 10).overlay(Circle().stroke(C.fondo, lineWidth: 3))
                        Rectangle().fill(C.menta).frame(height: 2)
                    }
                    .padding(.leading, 40).offset(y: y(m) - 5).zIndex(2)
                }
            }
        }
        .frame(height: y(hasta) + 8, alignment: .top)
        .padding(.top, 4)
    }

    private func bloqueTalon(_ t: Turno) -> some View {
        let est = t.visible(ahora)
        return Button { nav.turno = t } label: {
            HStack(spacing: 10) {
                Text(Formato.horaDe(t.inicio)).font(Tipo.lato(15, .black)).monospacedDigit()
                    .padding(.trailing, 10).frame(maxHeight: .infinity)
                    .overlay(alignment: .trailing) { Troquel() }
                VStack(alignment: .leading, spacing: 0) {
                    Text(t.cliente.nombre).font(Tipo.lato(15, .bold)).lineLimit(1)
                    Text(t.servicio).font(Tipo.lato(12)).foregroundStyle(C.sub).lineLimit(1)
                }
                Spacer(minLength: 0)
                Image(systemName: est.simbolo).font(.system(size: 18)).accessibilityLabel(est.texto)
            }
            .foregroundStyle(C.tinta).padding(.horizontal, 12)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(C.tarjeta, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            .overlay { if est == .enCurso { RoundedRectangle(cornerRadius: 14, style: .continuous).stroke(C.seleccion, lineWidth: 2) } }
            .shadow(color: Color(hex: 0x0F2328, alfa: 0.18), radius: 12, y: 8)
            .opacity(est == .atendido ? 0.55 : 1)
        }
        .buttonStyle(.plain)
    }

    // MARK: Todos: la pila del día de todas.

    @ViewBuilder private var todos: some View {
        let turnos = datos.turnos.filter { Formato.diaISO($0.inicio) == Formato.diaISO(dia) && $0.ocupa }.sorted { $0.inicio < $1.inicio }
        let huecos = dias.flatMap(\.huecos).sorted { $0.desde < $1.desde }
        if turnos.isEmpty && huecos.isEmpty {
            VacioQueEnsena(rayas: "Sin horario ese día", titulo: "Nadie atiende este día", texto: "Configurá los horarios de cada profesional desde Más.") { EmptyView() }
        }
        ForEach(turnos) { t in
            VistaTalon(turno: t, ahora: ahora, alTocar: { nav.turno = t },
                       alDeslizar: estado.soloLectura ? nil : { e in Task { await datos.cambiar(estado, t, a: e) } })
        }
        if !huecos.isEmpty {
            Rotulo("LIBRES")
            ForEach(huecos) { h in
                VistaHueco(hueco: h, accion: "Cargar", simboloAccion: "plus", alTocar: {
                    nav.nuevo = NuevoTurnoInicio(profesionalId: h.profesionalId, inicio: Dia.fecha(dia, minutos: h.desde))
                }, alAccion: {
                    nav.nuevo = NuevoTurnoInicio(profesionalId: h.profesionalId, inicio: Dia.fecha(dia, minutos: h.desde))
                })
            }
        }
    }

    // MARK: La cápsula de abajo: profesional anterior / siguiente, con los puntos (incluye «Todos»).

    private var capsula: some View {
        let total = dias.count + 1
        let nombre = elegida.map { "\($0.nombre.split(separator: " ").first ?? "") · \($0.ocupado.count) \($0.ocupado.count == 1 ? "turno" : "turnos")" } ?? "Todos"
        return HStack(spacing: 2) {
            Button { withAnimation(Movimiento.entrar) { indice = (indice - 1 + total) % total } } label: {
                Image(systemName: "chevron.left").font(.system(size: 20, weight: .semibold)).frame(width: 44, height: 44)
            }
            .accessibilityLabel("Profesional anterior")
            VStack(spacing: 3) {
                Text(nombre).font(Tipo.lato(15, .bold))
                HStack(spacing: 5) {
                    ForEach(0..<total, id: \.self) { i in
                        Capsule().fill(i == indice ? C.menta : .white.opacity(0.35)).frame(width: i == indice ? 16 : 5, height: 5)
                    }
                }
            }
            .frame(minWidth: 132)
            Button { withAnimation(Movimiento.entrar) { indice = (indice + 1) % total } } label: {
                Image(systemName: "chevron.right").font(.system(size: 20, weight: .semibold)).frame(width: 44, height: 44)
            }
            .accessibilityLabel("Profesional siguiente")
        }
        .foregroundStyle(.white).padding(.horizontal, 6).frame(height: 56)
        .background(C.petroleoHondo, in: Capsule())
        .shadow(color: .black.opacity(0.4), radius: 15, y: 14)
    }

    // MARK: 08 Semana: una tarjeta por día con la franja de cada profesional y las horas libres.

    private var vistaSemana: some View {
        ForEach(semana, id: \.self) { d in
            let delDia = Dia.armar(pros, datos.turnos, fecha: d, ahora: Formato.diaISO(d) == Formato.diaISO(Date()) ? Dia.minutosAhora() : nil)
            let (desde, hasta) = Dia.rango(delDia)
            let total = CGFloat(max(60, hasta - desde))
            let libres = delDia.reduce(0) { $0 + $1.libres }
            let on = Formato.diaISO(d) == Formato.diaISO(Date())
            if delDia.contains(where: { !$0.franjas.isEmpty }) {
                Button { dia = d; modo = "dia" } label: {
                    VStack(alignment: .leading, spacing: 8) {
                        HStack(alignment: .firstTextBaseline) {
                            Text(Formato.mayuscula(Formato.fechaLarga(d).components(separatedBy: " de ").first ?? "")).font(Tipo.lato(16, .bold))
                            Spacer()
                            HStack(spacing: 4) {
                                Rayas(quieto: true, paso: 5, color: C.menta, fondo: .clear).frame(width: 14, height: 10).clipShape(RoundedRectangle(cornerRadius: 3))
                                Text(Formato.libres(libres)).font(Tipo.lato(13, .bold)).monospacedDigit()
                            }
                            .foregroundStyle(C.mentaTexto)
                        }
                        ForEach(delDia) { p in
                            HStack(spacing: 8) {
                                Text(p.nombre.split(separator: " ").first.map(String.init) ?? p.nombre).font(Tipo.lato(12, .bold)).foregroundStyle(C.sub).frame(width: 44, alignment: .leading).lineLimit(1)
                                GeometryReader { g in
                                    ZStack(alignment: .leading) {
                                        RoundedRectangle(cornerRadius: 3).fill(C.chip)
                                        ForEach(p.franjas, id: \.self) { f in
                                            Rayas(quieto: true, paso: 6, color: C.menta, fondo: .clear).clipShape(RoundedRectangle(cornerRadius: 3))
                                                .frame(width: g.size.width * CGFloat(f.largo) / total).offset(x: g.size.width * CGFloat(f.desde - desde) / total)
                                        }
                                        ForEach(p.ocupado.indices, id: \.self) { i in
                                            let o = p.ocupado[i]
                                            RoundedRectangle(cornerRadius: 3).fill(C.ocupado)
                                                .frame(width: max(2, g.size.width * CGFloat(o.tramo.largo) / total - 2)).offset(x: g.size.width * CGFloat(o.tramo.desde - desde) / total)
                                        }
                                    }
                                }
                                .frame(height: 10)
                            }
                        }
                    }
                    .foregroundStyle(C.tinta).padding(14)
                    .tarjeta()
                    .overlay { if on { RoundedRectangle(cornerRadius: Radio.tarjeta, style: .continuous).strokeBorder(C.menta, lineWidth: 2) } }
                }
                .buttonStyle(.plain)
                .accessibilityLabel("\(Formato.fechaLarga(d)), \(Formato.libres(libres))")
            }
        }
    }
}
