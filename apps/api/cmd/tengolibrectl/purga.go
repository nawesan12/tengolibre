package main

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cuenta"
)

// purgaNegocios lista (y, sin listar, borra) los negocios cerrados hace más de 90 días, como el
// migrador. Borrar corre la misma función que el worker; las imágenes las borra el worker (acá no
// está la carpeta de archivos), así que conviene dejar que lo haga él.
func purgaNegocios(ctx context.Context, url string, out io.Writer, listar bool, ahora time.Time) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	ns, err := cuenta.ParaPurgar(ctx, conn, ahora)
	if err != nil {
		return err
	}
	EscribirParaPurgar(out, ns)
	if listar {
		return nil
	}
	var ids []string
	if err := conn.QueryRow(ctx, "SELECT app.purgar_negocios($1, 1000)::text[]", ahora.AddDate(0, 0, -cuenta.DiasNegocioCerrado)).Scan(&ids); err != nil {
		return err
	}
	fmt.Fprintf(out, "purgados: %d (las imágenes de %s/<id> quedan para borrar a mano)\n", len(ids), "TENGOLIBRE_ARCHIVOS")
	return nil
}

// EscribirParaPurgar: una tabla con lo que se borraría.
func EscribirParaPurgar(out io.Writer, ns []cuenta.NegocioParaPurgar) {
	if len(ns) == 0 {
		fmt.Fprintf(out, "no hay negocios cerrados hace más de %d días\n", cuenta.DiasNegocioCerrado)
		return
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSLUG\tNOMBRE\tCERRADO\tCLIENTES\tTURNOS\tCOBROS\t")
	retenidos := 0
	for _, n := range ns {
		estado := ""
		if n.Retenido {
			estado = "retenido: suscripción que cobra"
			retenidos++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%d\t%s\n", n.ID, n.Slug, n.Nombre, n.CerradoEn.Format(time.DateOnly), n.Clientes, n.Turnos, n.Cobros, estado)
	}
	_ = w.Flush()
	fmt.Fprintf(out, "%d negocios: %d se borrarían, %d retenidos (los cobros quedan archivados sin datos personales)\n",
		len(ns), len(ns)-retenidos, retenidos)
}
