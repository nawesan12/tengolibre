import SwiftUI
import UIKit

// Tengo Libre · los tokens del handoff de iPhone (design_handoff_tengolibre_ios/Theme.swift y los
// valores en línea de «App iPhone - Pantalla.dc.html»). Lato sola; horas y plata con
// .monospacedDigit(); rótulos en MAYÚSCULAS con tracking 0,08em. Sobre menta sólo va petróleo hondo.

extension Color {
    init(hex: UInt32, alfa: Double = 1) {
        self.init(.sRGB, red: Double((hex >> 16) & 0xFF) / 255, green: Double((hex >> 8) & 0xFF) / 255,
                  blue: Double(hex & 0xFF) / 255, opacity: alfa)
    }

    /// Un color «#rrggbb» que viene de la API (etiquetas, marca del negocio).
    init?(hexTexto: String) {
        let h = hexTexto.trimmingCharacters(in: .whitespaces).replacingOccurrences(of: "#", with: "")
        guard h.count == 6, let n = UInt32(h, radix: 16) else { return nil }
        self.init(hex: n)
    }
}

/// Los colores. Los de la marca no cambian con el modo; los demás salen del catálogo (claro y oscuro).
enum C {
    // Marca
    static let petroleo = Color(hex: 0x183C44)
    static let petroleoHondo = Color(hex: 0x102B31)
    static let menta = Color(hex: 0x3CCC90)
    static let crema = Color(hex: 0xF4EEE2)
    static let papel = Color(hex: 0xFFFDF9)
    // Tema (Assets.xcassets)
    static let fondo = Color("Fondo")
    static let tarjeta = Color("Tarjeta")
    static let tinta = Color("Tinta")
    static let sub = Color("Sub")
    static let linea = Color("Linea")
    static let chip = Color("Chip")
    static let cabecera = Color("Cabecera")
    static let mentaTexto = Color("MentaTexto")
    static let mentaSuave = Color("MentaSuave")
    static let huecoTinta = Color("HuecoTinta")
    static let huecoLinea = Color("HuecoLinea")
    /// Lo elegido: petróleo con texto blanco en claro, menta con petróleo hondo en oscuro.
    static let seleccion = Color("Seleccion")
    static let seleccionTexto = Color("SeleccionTexto")
    /// La hoja (detalle, nuevo turno): la tarjeta en los dos modos.
    static let hoja = tarjeta
    /// Los campos de texto: el fondo crema en claro, más hondo en oscuro.
    static let campo = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(Color(hex: 0x0F2025)) : UIColor(Color(hex: 0xF4EEE2)) })
    /// Los puntos del fondo crema (18 pt).
    static let punto = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(Color(hex: 0xF4EEE2, alfa: 0.05)) : UIColor(Color(hex: 0x183C44, alfa: 0.1)) })
    /// La barra de pestañas (94 % + blur).
    static let barra = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(Color(hex: 0x0E1E22, alfa: 0.94)) : UIColor(Color(hex: 0xFFFDF9, alfa: 0.94)) })
    /// Lo ocupado en «El día» de las tarjetas de la semana.
    static let ocupado = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(Color(hex: 0x5B8790)) : UIColor(Color(hex: 0x183C44)) })
    /// Tarjetas petróleo (membresía, sólo lectura): más claras en oscuro para que se despeguen.
    static let bloquePetroleo = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(Color(hex: 0x1F4A53)) : UIColor(Color(hex: 0x183C44)) })
    // Estados que no son de la marca (los de la web, estados.ts): siempre con ícono y palabra.
    static let pagoFondo = Color(hex: 0xFBE9C2)
    static let pagoTinta = Color(hex: 0x7A4A00)
    static let ausenteFondo = Color(hex: 0xFBE0DC)
    static let ausenteTinta = Color(hex: 0xA3241A)
    static let error = Color(hex: 0xB42318)
}

/// La letra: Lato (400/700/900), que escala con Dynamic Type.
enum Tipo {
    static func lato(_ tam: CGFloat, _ peso: Font.Weight = .regular, _ estilo: Font.TextStyle = .body) -> Font {
        let nombre = peso == .black ? "Lato-Black" : (peso == .bold || peso == .heavy || peso == .semibold) ? "Lato-Bold" : "Lato-Regular"
        return .custom(nombre, size: tam, relativeTo: estilo)
    }
    static let tituloGrande = lato(34, .black, .largeTitle)
    static let titulo = lato(22, .black, .title2)
    static let cuerpo = lato(17, .bold, .body)
    static let texto = lato(15, .regular, .subheadline)
    static let chico = lato(13, .regular, .footnote)
    static let rotulo = lato(12, .black, .caption)
    static let horaTalon = lato(19, .black, .headline)
}

enum Espacio { static let s1: CGFloat = 4, s2: CGFloat = 8, s3: CGFloat = 12, s4: CGFloat = 16, s6: CGFloat = 24, s8: CGFloat = 32, s12: CGFloat = 48 }
enum Radio { static let talon: CGFloat = 18, tarjeta: CGFloat = 20, bloque: CGFloat = 24, hoja: CGFloat = 16 }

/// El movimiento de la marca (con Reduce Motion, sin animar).
enum Movimiento {
    static let entrar = Animation.timingCurve(0.2, 0.7, 0.2, 1, duration: 0.45)
    static let tocar = Animation.timingCurve(0.2, 0.7, 0.2, 1, duration: 0.15)
    static let rayaHueco: Double = 3.0
}

enum Haptico {
    static func suave() { UIImpactFeedbackGenerator(style: .light).impactOccurred() }
    static func exito() { UINotificationFeedbackGenerator().notificationOccurred(.success) }
    static func error() { UINotificationFeedbackGenerator().notificationOccurred(.error) }
    static func seleccion() { UISelectionFeedbackGenerator().selectionChanged() }
}

// MARK: - Texto

extension View {
    /// Rótulo: MAYÚSCULAS, 12 pt Black, tracking 0,08em, en el gris del tema.
    func rotulo(_ color: Color = C.sub) -> some View {
        self.font(Tipo.rotulo).tracking(12 * 0.08).textCase(.uppercase).foregroundStyle(color)
    }
}

/// Un rótulo de sección («PARA ATENDER», «DESPUÉS»).
struct Rotulo: View {
    var texto: String
    var color: Color = C.sub
    init(_ texto: String, color: Color = C.sub) { self.texto = texto; self.color = color }
    var body: some View {
        Text(texto).rotulo(color).padding(.horizontal, 4).padding(.top, 8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityAddTraits(.isHeader)
    }
}

// MARK: - Fondos

/// El grano de las cabeceras de petróleo: ruido blanco al 11 % (180 pt, repetido).
enum Grano {
    static let imagen: UIImage = {
        let lado = 180
        var gen = SystemRandomNumberGenerator()
        let r = UIGraphicsImageRenderer(size: CGSize(width: lado, height: lado))
        return r.image { ctx in
            for y in 0..<lado {
                for x in 0..<lado where Int.random(in: 0..<100, using: &gen) < 55 {
                    let a = CGFloat(Double.random(in: 0...0.22, using: &gen))
                    ctx.cgContext.setFillColor(UIColor(white: 1, alpha: a).cgColor)
                    ctx.cgContext.fill(CGRect(x: x, y: y, width: 1, height: 1))
                }
            }
        }
    }()
}

extension View {
    /// Fondo de petróleo con grano (cabeceras, ingreso, elegir negocio).
    func fondoPetroleo(_ color: Color = C.cabecera, brillo: UnitPoint? = nil) -> some View {
        background {
            ZStack {
                color
                if let brillo {
                    RadialGradient(colors: [C.menta.opacity(0.22), .clear], center: brillo, startRadius: 0, endRadius: 320)
                }
                Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5)
            }
            .ignoresSafeArea()
        }
    }

    /// El fondo crema con puntos de 1,2 pt cada 18 pt.
    func fondoPuntos() -> some View {
        background {
            ZStack {
                C.fondo
                Canvas { ctx, tam in
                    var y: CGFloat = 9
                    while y < tam.height {
                        var x: CGFloat = 9
                        while x < tam.width {
                            ctx.fill(Path(ellipseIn: CGRect(x: x - 1.2, y: y - 1.2, width: 2.4, height: 2.4)), with: .color(C.punto))
                            x += 18
                        }
                        y += 18
                    }
                }
            }
            .ignoresSafeArea()
        }
    }

    /// Tarjeta: el fondo de tarjeta, radio 20 y la sombra suave del handoff.
    func tarjeta(_ radio: CGFloat = Radio.tarjeta, fondo: Color = C.tarjeta) -> some View {
        self.background(fondo, in: RoundedRectangle(cornerRadius: radio, style: .continuous))
            .clipShape(RoundedRectangle(cornerRadius: radio, style: .continuous))
            .shadow(color: Color(hex: 0x0F2328, alfa: 0.18), radius: 14, x: 0, y: 10)
    }

    /// Contorno de 1,5 pt en la línea del tema (botones secundarios, campos).
    func contorno(_ radio: CGFloat, color: Color = C.linea, ancho: CGFloat = 1.5) -> some View {
        overlay(RoundedRectangle(cornerRadius: radio, style: .continuous).strokeBorder(color, lineWidth: ancho))
    }
}

// MARK: - El hueco (tiempo libre)

/// Rayas a −45°, de 2 pt de menta al 42 % cada 10 pt, que corren con un ciclo de 3 s
/// (Canvas + TimelineView). Quietas con Reduce Motion o cuando `quieto`.
struct Rayas: View {
    var quieto = false
    var paso: CGFloat = 10
    var color: Color = C.menta.opacity(0.42)
    var fondo: Color = C.tarjeta.opacity(0.92)
    @Environment(\.accessibilityReduceMotion) private var reducir

    var body: some View {
        TimelineView(.animation(paused: quieto || reducir)) { tl in
            Canvas { ctx, tam in
                ctx.fill(Path(CGRect(origin: .zero, size: tam)), with: .color(fondo))
                let t = quieto || reducir ? 0 : tl.date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: Movimiento.rayaHueco) / Movimiento.rayaHueco
                let corrimiento = CGFloat(t) * paso * 2
                var x = -tam.height - paso * 2 + corrimiento
                while x < tam.width + paso {
                    var p = Path()
                    p.move(to: CGPoint(x: x, y: tam.height))
                    p.addLine(to: CGPoint(x: x + tam.height, y: 0))
                    ctx.stroke(p, with: .color(color), lineWidth: 2)
                    x += paso
                }
            }
        }
        .accessibilityHidden(true)
    }
}

extension View {
    /// Un bloque libre: rayas + contorno menta de 1,5 pt + letra del hueco.
    func hueco(_ radio: CGFloat = Radio.talon, quieto: Bool = false) -> some View {
        self.foregroundStyle(C.huecoTinta)
            .background { Rayas(quieto: quieto) }
            .clipShape(RoundedRectangle(cornerRadius: radio, style: .continuous))
            .contorno(radio, color: C.huecoLinea)
    }
}

/// La pastilla de papel que va sobre las rayas («LIBRE · 45 MIN»).
struct PastillaPapel<Contenido: View>: View {
    var radio: CGFloat = 999
    @ViewBuilder var contenido: Contenido
    var body: some View {
        contenido.padding(.horizontal, 8).padding(.vertical, 2)
            .background(C.tarjeta, in: RoundedRectangle(cornerRadius: radio, style: .continuous))
    }
}

// MARK: - Botones

/// La acción principal: pastilla menta, petróleo hondo, 900.
struct BotonMenta: ButtonStyle {
    var alto: CGFloat = 56
    var tam: CGFloat = 17
    var conSombra = true
    @Environment(\.isEnabled) private var habilitado
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(Tipo.lato(tam, .black)).monospacedDigit()
            .foregroundStyle(C.petroleoHondo)
            .frame(maxWidth: .infinity, minHeight: alto)
            .padding(.horizontal, 16)
            .background(C.menta, in: Capsule())
            .shadow(color: conSombra ? Color(hex: 0x17805A, alfa: 0.6) : .clear, radius: 13, y: 12)
            .opacity(habilitado ? (configuration.isPressed ? 0.85 : 1) : 0.45)
            .scaleEffect(configuration.isPressed ? 0.98 : 1)
            .animation(Movimiento.tocar, value: configuration.isPressed)
    }
}

/// Pastilla menta chica, del ancho del texto (Avisar, Responder, Reservar).
struct BotonMentaChico: ButtonStyle {
    var alto: CGFloat = 44
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(Tipo.lato(14, .black))
            .foregroundStyle(C.petroleoHondo)
            .padding(.horizontal, 16).frame(minHeight: alto)
            .background(C.menta, in: Capsule())
            .opacity(configuration.isPressed ? 0.85 : 1)
    }
}

/// Secundario: contorno de 1,5 pt, 700.
struct BotonContorno: ButtonStyle {
    var alto: CGFloat = 52
    var tam: CGFloat = 17
    var lleno = true
    @Environment(\.isEnabled) private var habilitado
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(Tipo.lato(tam, .bold))
            .foregroundStyle(C.tinta)
            .frame(maxWidth: lleno ? .infinity : nil, minHeight: alto)
            .padding(.horizontal, 16)
            .background(configuration.isPressed ? C.chip : .clear, in: Capsule())
            .contorno(999)
            .opacity(habilitado ? 1 : 0.45)
    }
}

/// Botón sobre fondo chip, radio 16 (Llamar, WhatsApp).
struct BotonChip: ButtonStyle {
    var alto: CGFloat = 52
    var radio: CGFloat = 16
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(Tipo.lato(16, .bold)).foregroundStyle(C.tinta)
            .frame(maxWidth: .infinity, minHeight: alto)
            .background(C.chip.opacity(configuration.isPressed ? 0.7 : 1), in: RoundedRectangle(cornerRadius: radio, style: .continuous))
    }
}

/// Botón redondo de 44 pt (acciones de la cabecera y de las hojas).
struct BotonRedondo: View {
    var simbolo: String
    var etiqueta: String
    var sobrePetroleo = false
    var numero: Int = 0
    var accion: () -> Void
    var body: some View {
        Button(action: accion) {
            Image(systemName: simbolo).font(.system(size: 19, weight: .semibold))
                .symbolEffect(.bounce, value: numero)
                .frame(width: 44, height: 44)
                .foregroundStyle(sobrePetroleo ? .white : C.tinta)
                .background(sobrePetroleo ? Color.white.opacity(0.1) : C.chip, in: Circle())
                .overlay(alignment: .topTrailing) {
                    if numero > 0 {
                        Text("\(numero)").font(Tipo.lato(11, .black)).monospacedDigit()
                            .foregroundStyle(C.petroleoHondo)
                            .padding(.horizontal, 5).frame(minWidth: 18, minHeight: 18)
                            .background(C.menta, in: Capsule())
                            .offset(x: 0, y: 2)
                    }
                }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(numero > 0 ? "\(etiqueta), \(numero) sin leer" : etiqueta)
    }
}

/// Botón de texto menta (Cancelar en las hojas, «Ver lo último guardado»).
struct BotonTexto: ButtonStyle {
    var color: Color = C.mentaTexto
    func makeBody(configuration: Configuration) -> some View {
        configuration.label.font(Tipo.lato(15, .bold)).foregroundStyle(color)
            .frame(minHeight: 44).contentShape(Rectangle())
            .opacity(configuration.isPressed ? 0.6 : 1)
    }
}

// MARK: - Piezas

/// Iniciales en un círculo chip.
struct Iniciales: View {
    var nombre: String
    var tam: CGFloat = 44
    var fondo: Color = C.chip
    var tinta: Color = C.tinta
    var radio: CGFloat? = nil
    var body: some View {
        let ini = nombre.split(separator: " ").prefix(2).compactMap { $0.first }.map(String.init).joined().uppercased()
        Text(ini.isEmpty ? "·" : ini).font(Tipo.lato(tam * 0.34, .black))
            .foregroundStyle(tinta)
            .frame(width: tam, height: tam)
            .background(fondo, in: RoundedRectangle(cornerRadius: radio ?? tam / 2, style: .continuous))
            .accessibilityHidden(true)
    }
}

/// Chip con ícono y palabra (etiquetas de clienta, filtros).
struct Chip: View {
    var texto: String
    var simbolo: String? = nil
    var elegido = false
    var alto: CGFloat = 36
    var body: some View {
        HStack(spacing: 4) {
            if let simbolo { Image(systemName: simbolo).font(.system(size: 15, weight: .semibold)) }
            Text(texto).font(Tipo.lato(14, .bold)).lineLimit(1)
        }
        .padding(.leading, simbolo == nil ? 14 : 8).padding(.trailing, 14)
        .frame(height: alto)
        .foregroundStyle(elegido ? C.seleccionTexto : C.tinta)
        .background(elegido ? C.seleccion : Color.clear, in: Capsule())
        .contorno(999, color: elegido ? .clear : C.linea)
    }
}

/// Segmentado del handoff: sobre petróleo (cabeceras) o sobre la tarjeta.
struct Segmentado<V: Hashable>: View {
    var opciones: [(V, String)]
    @Binding var elegido: V
    var sobrePetroleo = true
    var body: some View {
        HStack(spacing: 4) {
            ForEach(opciones, id: \.0) { v, t in
                let on = v == elegido
                Button {
                    Haptico.seleccion()
                    withAnimation(Movimiento.tocar) { elegido = v }
                } label: {
                    Text(t).font(Tipo.lato(14, .bold)).lineLimit(1).minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity, minHeight: 36)
                        .foregroundStyle(on ? C.petroleoHondo : (sobrePetroleo ? .white : C.tinta))
                        .background(on ? C.menta : .clear, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(on ? .isSelected : [])
            }
        }
        .padding(4)
        .background(sobrePetroleo ? Color.white.opacity(0.1) : C.chip, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

/// Un campo de texto de 16 pt (mínimo del handoff) sobre el fondo de campo.
struct Campo: View {
    var titulo: String
    @Binding var texto: String
    var ayuda: String = ""
    var teclado: UIKeyboardType = .default
    var multilinea = false
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            if !titulo.isEmpty { Text(titulo).rotulo() }
            Group {
                if multilinea {
                    TextField(ayuda, text: $texto, axis: .vertical).lineLimit(3...8)
                } else {
                    TextField(ayuda, text: $texto)
                }
            }
            .keyboardType(teclado)
            .textInputAutocapitalization(teclado == .emailAddress || teclado == .URL ? .never : .sentences)
            .autocorrectionDisabled(teclado != .default)
            .font(Tipo.lato(16)).foregroundStyle(C.tinta)
            .padding(.horizontal, 14).padding(.vertical, 13)
            .background(C.campo, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
    }
}

/// Una fila de lista (Más, ajustes): ícono en cuadradito chip, texto, detalle y chevron.
struct FilaLista: View {
    var simbolo: String
    var titulo: String
    var detalle: String = ""
    var final: String = "chevron.right"
    var tinta: Color = C.tinta
    /// Algo para hacer ahí (Mercado Pago sin conectar, IA apagada…): punto menta y el detalle en menta.
    var pendiente = false
    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: simbolo).font(.system(size: 17, weight: .semibold))
                .frame(width: 32, height: 32)
                .background(C.chip, in: RoundedRectangle(cornerRadius: 9, style: .continuous))
                .foregroundStyle(tinta)
                .overlay(alignment: .topTrailing) {
                    if pendiente { Circle().fill(C.menta).frame(width: 10, height: 10).overlay(Circle().stroke(C.tarjeta, lineWidth: 2)).offset(x: 3, y: -3) }
                }
            VStack(alignment: .leading, spacing: 1) {
                Text(titulo).font(Tipo.lato(16)).foregroundStyle(tinta)
                if !detalle.isEmpty { Text(detalle).font(Tipo.lato(12, pendiente ? .bold : .regular)).foregroundStyle(pendiente ? C.mentaTexto : C.sub) }
            }
            Spacer(minLength: 0)
            if !final.isEmpty { Image(systemName: final).font(.system(size: 15, weight: .semibold)).foregroundStyle(C.sub) }
        }
        .padding(.horizontal, 14).frame(minHeight: 56)
        .contentShape(Rectangle())
    }
}

/// Una tarjeta-lista: filas separadas por la línea del tema.
struct ListaTarjeta<Item: Identifiable, Fila: View>: View {
    var items: [Item]
    @ViewBuilder var fila: (Item) -> Fila
    var body: some View {
        VStack(spacing: 0) {
            ForEach(Array(items.enumerated()), id: \.element.id) { i, it in
                if i > 0 { Rectangle().fill(C.linea).frame(height: 1) }
                fila(it)
            }
        }
        .tarjeta()
    }
}

/// El interruptor del handoff (51×31, menta encendido).
struct Interruptor: View {
    var titulo: String
    var detalle: String = ""
    var simbolo: String? = nil
    @Binding var activo: Bool
    var body: some View {
        Toggle(isOn: $activo) {
            HStack(spacing: 12) {
                if let simbolo { Image(systemName: simbolo).font(.system(size: 20, weight: .semibold)).foregroundStyle(C.tinta) }
                VStack(alignment: .leading, spacing: 2) {
                    Text(titulo).font(Tipo.lato(16)).foregroundStyle(C.tinta)
                    if !detalle.isEmpty { Text(detalle).font(Tipo.lato(12)).foregroundStyle(C.sub) }
                }
            }
        }
        .tint(C.menta)
        .frame(minHeight: 56)
    }
}

/// Esqueleto de carga (shimmer de la tarjeta al chip).
struct Esqueleto: View {
    var alto: CGFloat = 80
    var radio: CGFloat = 18
    @State private var fase: CGFloat = -1
    @Environment(\.accessibilityReduceMotion) private var reducir
    var body: some View {
        RoundedRectangle(cornerRadius: radio, style: .continuous)
            .fill(LinearGradient(colors: [C.tarjeta, C.chip, C.tarjeta], startPoint: UnitPoint(x: fase, y: 0.5), endPoint: UnitPoint(x: fase + 1, y: 0.5)))
            .frame(height: alto)
            .onAppear { if !reducir { withAnimation(.linear(duration: 1.4).repeatForever(autoreverses: false)) { fase = 1 } } }
            .accessibilityHidden(true)
    }
}

/// Vacío que enseña: rayas, título, texto y acciones (handoff 20).
struct VacioQueEnsena<Acciones: View>: View {
    var rayas: String
    var titulo: String
    var texto: String
    @ViewBuilder var acciones: Acciones
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            PastillaPapel { Text(rayas).font(Tipo.lato(15, .black)).foregroundStyle(C.huecoTinta).padding(.vertical, 6).padding(.horizontal, 6) }
                .frame(maxWidth: .infinity, minHeight: 150)
                .hueco()
            Text(titulo).font(Tipo.lato(22, .black)).tracking(-0.44).foregroundStyle(C.tinta).fixedSize(horizontal: false, vertical: true)
            Text(texto).font(Tipo.lato(15)).foregroundStyle(C.sub).lineSpacing(3).fixedSize(horizontal: false, vertical: true)
            acciones
        }
        .padding(20).tarjeta(Radio.bloque)
    }
}

/// Error con salida (handoff 22).
struct ErrorConSalida: View {
    var titulo: String
    var texto: String
    var reintentar: () -> Void
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "arrow.triangle.2.circlepath").font(.system(size: 28, weight: .semibold))
                .frame(width: 64, height: 64).background(C.chip, in: Circle())
            Text(titulo).font(Tipo.lato(20, .bold)).foregroundStyle(C.tinta).multilineTextAlignment(.center)
            Text(texto).font(Tipo.lato(15)).foregroundStyle(C.sub).multilineTextAlignment(.center)
            Button("Reintentar", action: reintentar).buttonStyle(BotonMenta(alto: 52, conSombra: false))
        }
        .padding(.vertical, 32).padding(.horizontal, 20).frame(maxWidth: .infinity).tarjeta(Radio.bloque)
    }
}

/// Un aviso arriba de todo (toast del handoff): petróleo hondo, ícono en círculo menta y una acción.
struct Toast: Equatable {
    var simbolo: String
    var titulo: String
    var texto: String = ""
    var accion: String = ""
    var error = false
    var id = UUID()
}

struct VistaToast: View {
    var t: Toast
    var alTocar: () -> Void
    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: t.simbolo).font(.system(size: 19, weight: .bold))
                .foregroundStyle(t.error ? .white : C.petroleoHondo)
                .frame(width: 40, height: 40)
                .background(t.error ? C.error : C.menta, in: Circle())
            VStack(alignment: .leading, spacing: 2) {
                Text(t.titulo).font(Tipo.lato(15, .bold))
                if !t.texto.isEmpty { Text(t.texto).font(Tipo.lato(13)).foregroundStyle(.white.opacity(0.78)) }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if !t.accion.isEmpty {
                Button(t.accion, action: alTocar).font(Tipo.lato(15, .bold)).foregroundStyle(.white)
                    .padding(.horizontal, 16).frame(minHeight: 44).background(.white.opacity(0.12), in: Capsule())
            }
        }
        .foregroundStyle(.white)
        .padding(.leading, 14).padding([.vertical, .trailing], 12)
        .background(C.petroleoHondo, in: RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: .black.opacity(0.35), radius: 20, y: 14)
        .padding(.horizontal, 12)
        .accessibilityElement(children: .combine)
        .accessibilityAddTraits(.isStaticText)
    }
}

/// La pastilla de un estado: siempre ícono + palabra.
struct PastillaEstado: View {
    var texto: String
    var simbolo: String
    var fondo: Color
    var tinta: Color
    var alto: CGFloat = 24
    var body: some View {
        HStack(spacing: 4) {
            Image(systemName: simbolo).font(.system(size: alto * 0.55, weight: .bold))
            Text(texto).font(Tipo.lato(alto * 0.5, .bold)).lineLimit(1)
        }
        .padding(.leading, 6).padding(.trailing, 9).frame(height: alto)
        .foregroundStyle(tinta).background(fondo, in: Capsule())
    }
}
