// Package privado: lo que se escribe en los logs sin datos personales. Los logs de Docker no tienen la
// retención que declara la política: un rechazo 550 de SMTP trae la dirección del destinatario, un
// error de WhatsApp el número y el de Mercado Pago el mail de quien pagó. El texto completo queda
// donde sí se purga (ultimo_error de cada aviso).
package privado

import "regexp"

var (
	reEmail   = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@([A-Za-z0-9.\-]+\.[A-Za-z]{2,})`)
	reDigitos = regexp.MustCompile(`\+?\d[\d\s\-]{6,}\d`)
	reFecha   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Enmascarar: los emails pasan a ***@dominio y las secuencias de 8 o más dígitos (teléfonos,
// documentos) dejan sólo los últimos 4.
func Enmascarar(s string) string {
	s = reEmail.ReplaceAllString(s, "***@$1")
	return reDigitos.ReplaceAllStringFunc(s, func(x string) string {
		if reFecha.MatchString(x) {
			return x
		}
		d := make([]rune, 0, len(x))
		for _, r := range x {
			if r >= '0' && r <= '9' {
				d = append(d, r)
			}
		}
		if len(d) < 8 {
			return x
		}
		return "…" + string(d[len(d)-4:])
	})
}

// Error enmascarado (nil: "").
func Error(err error) string {
	if err == nil {
		return ""
	}
	return Enmascarar(err.Error())
}
