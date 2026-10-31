// Comando vapid: genera un par de claves VAPID para el push web del panel
// (TENGOLIBRE_VAPID_PUBLICA y TENGOLIBRE_VAPID_PRIVADA). Se corre una vez; cambiarlas deja sin
// avisos a los navegadores ya suscriptos hasta que vuelvan a activarlos.
package main

import (
	"fmt"
	"os"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/webpush"
)

func main() {
	pub, priv, err := webpush.GenerarClaves()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("TENGOLIBRE_VAPID_PUBLICA=%s\nTENGOLIBRE_VAPID_PRIVADA=%s\n", pub, priv)
}
