package asistente

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// El motor contesta lo que entendió el intérprete con las mismas operaciones del MCP (y el mismo
// rol de quien pregunta). Las acciones devuelven una propuesta.

// leerComo corre una herramienta de lectura y pasa el resultado a v.
func leerComo(ctx context.Context, op *mcp.Operador, nombre string, args map[string]any, v any) error {
	b, _ := json.Marshal(args)
	res, err := op.Leer(ctx, nombre, b)
	if err != nil {
		return err
	}
	j, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, v)
}

func (s *Servicio) motor(ctx context.Context, op *mcp.Operador, c Catalogo, in Intencion) (Respuesta, error) {
	hoy := c.hoy()
	f := in.Fecha
	if f.IsZero() {
		f = hoy
	}
	switch in.Tipo {
	case Agenda:
		ts, err := agendaDe(ctx, op, f, f, in.ProfesionalID)
		if err != nil {
			return Respuesta{}, err
		}
		r := Respuesta{Respuesta: textoAgenda(ts, f, hoy)}
		if len(ts) > 0 {
			r.Encabezado, r.Items = encabezadoAgenda(ts, f, hoy), items(ts, 25)
		}
		return r, nil
	case Quien:
		ts, err := agendaDe(ctx, op, f, f, in.ProfesionalID)
		if err != nil {
			return Respuesta{}, err
		}
		h := c.Resolver(in.Hora, f)
		r := Respuesta{Respuesta: textoQuien(ts, h, f, hoy)}
		if en := enCurso(ts, h); len(en) > 0 {
			r.Encabezado, r.Items = mayuscula(elDia(f, hoy)+" a las "+h.Format("15:04"))+":", items(en, 25)
		}
		return r, nil
	case Libres:
		return s.libres(ctx, op, c, in, f, hoy)
	case Mover, Cancelar, Confirmar:
		return s.accion(ctx, op, c, in)
	case Anotar:
		return s.anotar(ctx, op, c, in)
	}
	return Respuesta{}, problema.Invalido("No entendí el pedido.")
}

func agendaDe(ctx context.Context, op *mcp.Operador, desde, hasta time.Time, profesional string) ([]mcp.TurnoMCP, error) {
	ts, _, err := agendaCon(ctx, op, desde, hasta, profesional, "", false)
	return ts, err
}

// agendaCon: la agenda con los filtros que van a la base (cliente y sólo los que no empezaron), y si
// quedaron turnos afuera del tope.
func agendaCon(ctx context.Context, op *mcp.Operador, desde, hasta time.Time, profesional, cliente string, proximos bool) ([]mcp.TurnoMCP, bool, error) {
	args := map[string]any{"desde": desde.Format(time.DateOnly), "hasta": hasta.Format(time.DateOnly)}
	if profesional != "" {
		args["profesionalId"] = profesional
	}
	if cliente != "" {
		args["cliente"] = cliente
	}
	if proximos {
		args["soloProximos"] = true
	}
	var out struct {
		Turnos   []mcp.TurnoMCP
		Truncado bool
	}
	err := leerComo(ctx, op, "ver_agenda", args, &out)
	return out.Turnos, out.Truncado, err
}

// elDia: «hoy», «mañana» o «el jueves 20 de noviembre».
func elDia(f, hoy time.Time) string {
	switch {
	case f.Equal(hoy):
		return "hoy"
	case f.Equal(hoy.AddDate(0, 0, 1)):
		return "mañana"
	}
	return "el " + avisos.Fecha(f)
}

func mayuscula(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

func inicioDe(t mcp.TurnoMCP) time.Time {
	x, _ := time.Parse(time.RFC3339, t.Inicio)
	return x
}

func finDe(t mcp.TurnoMCP) time.Time {
	x, _ := time.Parse(time.RFC3339, t.Fin)
	return x
}

func linea(t mcp.TurnoMCP) string {
	l := "• " + inicioDe(t).Format("15:04") + " " + t.Cliente.Nombre + " — " + t.Servicio
	if t.Profesional != "" {
		l += " con " + t.Profesional
	}
	switch t.Estado {
	case "pendiente":
		l += " (sin confirmar)"
	case "pendiente_pago":
		l += " (falta la seña)"
	case "atendido":
		l += " (atendido)"
	case "ausente":
		l += " (no vino)"
	}
	return l
}

// estadoDicho: el estado de un turno en palabras para una lista (confirmado: nada).
func estadoDicho(e string) string {
	return map[string]string{"pendiente": "sin confirmar", "pendiente_pago": "falta la seña", "atendido": "atendido", "ausente": "no vino"}[e]
}

func items(ts []mcp.TurnoMCP, n int) []ItemTurno {
	out := []ItemTurno{}
	for i, t := range ts {
		if i == n {
			break
		}
		out = append(out, ItemTurno{ID: t.ID, Dia: t.Dia, Hora: inicioDe(t).Format("15:04"), Cliente: t.Cliente.Nombre,
			Servicio: t.Servicio, Profesional: t.Profesional, Estado: estadoDicho(t.Estado)})
	}
	return out
}

func encabezadoAgenda(ts []mcp.TurnoMCP, f, hoy time.Time) string {
	cuando := mayuscula(elDia(f, hoy))
	if len(ts) == 1 {
		return cuando + " hay 1 turno:"
	}
	return fmt.Sprintf("%s hay %d turnos:", cuando, len(ts))
}

// enCurso: los turnos que están en curso a la hora h.
func enCurso(ts []mcp.TurnoMCP, h time.Time) []mcp.TurnoMCP {
	var en []mcp.TurnoMCP
	for _, t := range ts {
		if !h.Before(inicioDe(t)) && h.Before(finDe(t)) {
			en = append(en, t)
		}
	}
	return en
}

func textoAgenda(ts []mcp.TurnoMCP, f, hoy time.Time) string {
	cuando := mayuscula(elDia(f, hoy))
	if len(ts) == 0 {
		return cuando + " no hay turnos."
	}
	var b strings.Builder
	if len(ts) == 1 {
		b.WriteString(cuando + " hay 1 turno:")
	} else {
		fmt.Fprintf(&b, "%s hay %d turnos:", cuando, len(ts))
	}
	for i, t := range ts {
		if i == 25 {
			fmt.Fprintf(&b, "\n… y %d más.", len(ts)-25)
			break
		}
		b.WriteString("\n" + linea(t))
	}
	return b.String()
}

func textoQuien(ts []mcp.TurnoMCP, h, f, hoy time.Time) string {
	var en []mcp.TurnoMCP
	var siguiente *mcp.TurnoMCP
	for i, t := range ts {
		i0, i1 := inicioDe(t), finDe(t)
		if !h.Before(i0) && h.Before(i1) {
			en = append(en, t)
		} else if i0.After(h) && siguiente == nil {
			siguiente = &ts[i]
		}
	}
	cuando := elDia(f, hoy) + " a las " + h.Format("15:04")
	if len(en) == 0 {
		x := mayuscula(cuando) + " no viene nadie."
		if siguiente != nil {
			x += " El siguiente es " + siguiente.Cliente.Nombre + " a las " + inicioDe(*siguiente).Format("15:04") + " (" + siguiente.Servicio + ")."
		}
		return x
	}
	var b strings.Builder
	b.WriteString(mayuscula(cuando) + ":")
	for _, t := range en {
		b.WriteString("\n" + linea(t))
	}
	return b.String()
}

// libres: si no dijo el servicio, el único que hay o el más corto (y se aclara).
func (s *Servicio) libres(ctx context.Context, op *mcp.Operador, c Catalogo, in Intencion, f, hoy time.Time) (Respuesta, error) {
	if len(c.Servicios) == 0 {
		return Respuesta{Respuesta: "Todavía no hay servicios cargados."}, nil
	}
	srv := ServicioCat{}
	for _, x := range c.Servicios {
		if x.ID == in.ServicioID {
			srv = x
		}
	}
	aclarar := false
	if srv.ID == "" {
		srv = c.Servicios[0]
		for _, x := range c.Servicios[1:] {
			if x.Duracion < srv.Duracion {
				srv = x
			}
		}
		aclarar = len(c.Servicios) > 1
	}
	args := map[string]any{"servicioId": srv.ID, "fecha": f.Format(time.DateOnly)}
	if in.ProfesionalID != "" {
		args["profesionalId"] = in.ProfesionalID
	}
	var out struct {
		Dias []struct {
			Horarios []mcp.LibreMCP `json:"horarios"`
		} `json:"dias"`
	}
	if err := leerComo(ctx, op, "buscar_horarios_libres", args, &out); err != nil {
		return Respuesta{}, err
	}
	var horas []string
	if len(out.Dias) > 0 {
		for _, h := range out.Dias[0].Horarios {
			mm := minutosDe(h.Hora)
			if EnFranja(in.Franja, mm) {
				horas = append(horas, h.Hora)
			}
		}
	}
	cuando := elDia(f, hoy)
	switch in.Franja {
	case "manana":
		cuando += " a la mañana"
	case "tarde":
		cuando += " a la tarde"
	case "noche":
		cuando += " a la noche"
	}
	para := " para " + srv.Nombre
	if aclarar {
		para += fmt.Sprintf(" (%d min; si es otro servicio, decime cuál)", srv.Duracion)
	}
	if len(horas) == 0 {
		return Respuesta{Respuesta: mayuscula(cuando) + " no hay lugar" + para + "."}, nil
	}
	if len(horas) > 16 {
		horas = append(horas[:16], "…")
	}
	return Respuesta{Respuesta: mayuscula(cuando) + " hay lugar" + para + " a las " + strings.Join(horas, ", ") + "."}, nil
}

var activos = []string{"pendiente", "confirmado", "pendiente_pago"}

// accion: busca el turno del que habla y arma la propuesta.
func (s *Servicio) accion(ctx context.Context, op *mcp.Operador, c Catalogo, in Intencion) (Respuesta, error) {
	hoy := c.hoy()
	desde, hasta := hoy, hoy.AddDate(0, 0, 30)
	if !in.Fecha.IsZero() {
		desde, hasta = in.Fecha, in.Fecha
	} else if in.Cliente == "" {
		// «Cancelá el de las 3»: el de hoy.
		hasta = hoy
	}
	ts, truncado, err := agendaCon(ctx, op, desde, hasta, "", in.Cliente, true)
	if err != nil {
		return Respuesta{}, err
	}
	ahora := c.Ahora
	var cands []mcp.TurnoMCP
	for _, t := range ts {
		i0 := inicioDe(t)
		if !contiene(activos, t.Estado) || i0.Before(ahora) {
			continue
		}
		if in.Tipo == Confirmar && !turnos.PuedePasar(t.Estado, "confirmado") || in.Tipo == Confirmar && t.Estado == "confirmado" {
			continue
		}
		if in.Cliente != "" && !coincideNombre(in.Cliente, t.Cliente.Nombre) {
			continue
		}
		if in.Hora.Dicha {
			dia := time.Date(i0.Year(), i0.Month(), i0.Day(), 0, 0, 0, 0, c.Ahora.Location())
			if !c.Resolver(in.Hora, dia).Equal(i0.In(c.Ahora.Location())) {
				continue
			}
		}
		cands = append(cands, t)
	}
	verbo := map[string]string{Mover: "mover", Cancelar: "cancelar", Confirmar: "confirmar"}[in.Tipo]
	if len(cands) == 0 && truncado {
		return Respuesta{Respuesta: "Hay demasiados turnos para buscar así. Decime también el día o el apellido."}, nil
	}
	if len(cands) == 0 {
		x := "No encontré un turno"
		if in.Cliente != "" {
			x += " de " + mayuscula(in.Cliente)
		}
		if in.Hora.Dicha {
			x += " a las " + c.Resolver(in.Hora, desde).Format("15:04")
		}
		if !in.Fecha.IsZero() || in.Cliente == "" {
			x += " " + elDia(desde, hoy)
		} else {
			x += " en los próximos 30 días"
		}
		if in.Tipo == Confirmar {
			x += " para confirmar (o ya está confirmado)"
		} else {
			x += " para " + verbo
		}
		return Respuesta{Respuesta: x + "."}, nil
	}
	if len(cands) > 1 {
		var b strings.Builder
		fmt.Fprintf(&b, "Encontré %d turnos. ¿Cuál querés %s? Decime el día y la hora:", len(cands), verbo)
		for i, t := range cands {
			if i == 6 {
				b.WriteString("\n…")
				break
			}
			b.WriteString("\n• " + elDia(dia0(inicioDe(t), c), hoy) + " a las " + inicioDe(t).In(c.Ahora.Location()).Format("15:04") +
				" " + t.Cliente.Nombre + " — " + t.Servicio)
		}
		return Respuesta{Respuesta: b.String()}, nil
	}
	t := cands[0]
	var nombre string
	args := map[string]any{"turnoId": t.ID}
	switch in.Tipo {
	case Cancelar:
		nombre = "cancelar_turno"
	case Confirmar:
		nombre = "confirmar_turno"
	case Mover:
		nombre = "reprogramar_turno"
		i0 := inicioDe(t).In(c.Ahora.Location())
		destino := in.Destino
		if destino.IsZero() {
			destino = dia0(i0, c)
		}
		nuevo := time.Date(destino.Year(), destino.Month(), destino.Day(), i0.Hour(), i0.Minute(), 0, 0, c.Ahora.Location())
		if in.DestinoHora.Dicha {
			nuevo = c.Resolver(in.DestinoHora, destino)
		}
		args["fecha"], args["hora"] = nuevo.Format(time.DateOnly), nuevo.Format("15:04")
	}
	raw, _ := json.Marshal(args)
	resumen, final, err := op.Previa(ctx, nombre, raw)
	if err != nil {
		var p *problema.Problema
		if in.Tipo == Mover && errors.As(err, &p) && p.Codigo == "horario_ocupado" {
			return Respuesta{Respuesta: mcp.Explicar(err, nombre) + s.alternativas(ctx, op, c, t, args["fecha"].(string), args["hora"].(string))}, nil
		}
		return Respuesta{}, err
	}
	p, err := s.proponer(ctx, nombre, resumen, final)
	if err != nil {
		return Respuesta{}, err
	}
	return Respuesta{Respuesta: "Te lo dejo listo para confirmar:", Propuestas: []Propuesta{p}}, nil
}

// alternativas: los horarios libres más cercanos al pedido, ese día y para el mismo servicio.
func (s *Servicio) alternativas(ctx context.Context, op *mcp.Operador, c Catalogo, t mcp.TurnoMCP, fecha, hora string) string {
	id := ""
	for _, x := range c.Servicios {
		if x.Nombre == t.Servicio {
			id = x.ID
		}
	}
	if id == "" {
		return ""
	}
	var out struct {
		Dias []struct {
			Horarios []mcp.LibreMCP `json:"horarios"`
		} `json:"dias"`
	}
	if leerComo(ctx, op, "buscar_horarios_libres", map[string]any{"servicioId": id, "fecha": fecha}, &out) != nil || len(out.Dias) == 0 {
		return ""
	}
	var hs []string
	for _, h := range out.Dias[0].Horarios {
		hs = append(hs, h.Hora)
	}
	if len(hs) == 0 {
		return " Ese día no queda lugar para " + t.Servicio + "."
	}
	pedido := minutosDe(hora)
	lejos := func(h string) int {
		d := minutosDe(h) - pedido
		if d < 0 {
			return -d
		}
		return d
	}
	sort.SliceStable(hs, func(i, j int) bool { return lejos(hs[i]) < lejos(hs[j]) })
	if len(hs) > 6 {
		hs = hs[:6]
	}
	sort.Strings(hs)
	return " Ese día hay lugar para " + t.Servicio + " a las " + strings.Join(hs, ", ") + "."
}

func dia0(t time.Time, c Catalogo) time.Time {
	t = t.In(c.Ahora.Location())
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, c.Ahora.Location())
}

func contiene(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// anotar: «anotá a Lucía mañana a las 11 para corte». Sólo con una clienta que ya existe (por su
// nombre, si hay una sola); una nueva necesita su teléfono y se carga en «Nuevo turno». Sin servicio,
// pregunta cuál. Vuelve como propuesta, como las demás acciones.
func (s *Servicio) anotar(ctx context.Context, op *mcp.Operador, c Catalogo, in Intencion) (Respuesta, error) {
	if in.ServicioID == "" {
		if len(c.Servicios) == 1 {
			in.ServicioID = c.Servicios[0].ID
		} else {
			var ns []string
			for i, x := range c.Servicios {
				if i == 6 {
					break
				}
				ns = append(ns, x.Nombre)
			}
			return Respuesta{Respuesta: "¿Para qué servicio? Decímelo así: «anotá a " + mayuscula(in.Cliente) + " … para " +
				strings.ToLower(ns[0]) + "». Tenés " + strings.Join(ns, ", ") + "."}, nil
		}
	}
	nuevo := "No encontré a " + mayuscula(in.Cliente) + " entre tus clientes. Si es la primera vez, cargá el turno desde «Nuevo turno» con su\u00a0teléfono."
	if !op.Puede("buscar_clientes") {
		return Respuesta{Respuesta: "Ese turno cargalo desde «Nuevo turno»: ahí van el nombre y el\u00a0teléfono."}, nil
	}
	var out struct {
		Clientes []mcp.ClienteMCP `json:"clientes"`
	}
	if err := leerComo(ctx, op, "buscar_clientes", map[string]any{"texto": in.Cliente, "limite": 5}, &out); err != nil {
		return Respuesta{}, err
	}
	var cands []mcp.ClienteMCP
	for _, x := range out.Clientes {
		if coincideNombre(in.Cliente, x.Nombre) {
			cands = append(cands, x)
		}
	}
	switch {
	case len(cands) == 0:
		return Respuesta{Respuesta: nuevo}, nil
	case len(cands) > 1:
		var b strings.Builder
		b.WriteString("Hay varias personas con ese nombre. ¿Cuál? Decime el apellido:")
		for _, x := range cands {
			b.WriteString("\n• " + x.Nombre)
		}
		return Respuesta{Respuesta: b.String()}, nil
	}
	inicio := c.Resolver(in.Hora, in.Fecha)
	args := map[string]any{"servicioId": in.ServicioID, "clienteId": cands[0].ID, "fecha": inicio.Format(time.DateOnly), "hora": inicio.Format("15:04")}
	if in.ProfesionalID != "" {
		args["profesionalId"] = in.ProfesionalID
	}
	raw, _ := json.Marshal(args)
	resumen, final, err := op.Previa(ctx, "crear_turno", raw)
	if err != nil {
		return Respuesta{}, err
	}
	p, err := s.proponer(ctx, "crear_turno", resumen, final)
	if err != nil {
		return Respuesta{}, err
	}
	return Respuesta{Respuesta: "Te lo dejo listo para confirmar:", Propuestas: []Propuesta{p}}, nil
}
