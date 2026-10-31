package whatsapp

import (
	"errors"
	"strings"
)

// ErrTelefono: el teléfono no se puede llevar a E.164.
var ErrTelefono = errors.New("whatsapp: teléfono inválido")

// E164 normaliza el teléfono como lo tipeó el cliente («11 15-2345-6789», «0351 155 123456»,
// «+54 9 11 2345 6789») a E.164. Sin código de país se toma Argentina, y un número argentino se
// lleva a la forma de celular que usa WhatsApp: +54 9 + característica + número (10 dígitos, sin el
// 0 ni el 15). Un número con + de otro país se deja como viene (8 a 15 dígitos).
func E164(tel string) (string, error) {
	tel = strings.TrimSpace(tel)
	internacional := strings.HasPrefix(tel, "+")
	var b strings.Builder
	for _, r := range tel {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if !internacional && strings.HasPrefix(d, "00") {
		internacional, d = true, d[2:]
	}
	// «549…» o «54 11…» sin el +: también es internacional (un número argentino nacional tiene 10
	// dígitos, o 11-13 con el 0 y el 15, y nunca empieza con 54 seguido de tantos).
	if !internacional && strings.HasPrefix(d, "54") && len(d) >= 12 && !strings.HasPrefix(d, "0") {
		internacional = true
	}
	if internacional {
		if !strings.HasPrefix(d, "54") {
			if len(d) < 8 || len(d) > 15 || d[0] == '0' {
				return "", ErrTelefono
			}
			return "+" + d, nil
		}
		d = strings.TrimPrefix(d[2:], "9")
	}
	n, err := nacional(strings.TrimPrefix(d, "0"))
	if err != nil {
		return "", err
	}
	return "+549" + n, nil
}

// nacional deja característica + número (10 dígitos) sacando el 15 del celular si está.
func nacional(s string) (string, error) {
	switch len(s) {
	case 10:
		// Ninguna característica empieza con 0 ni con 15: «15 2345 6789» es un celular sin
		// característica, y no se puede adivinar de dónde es.
		if s[0] == '0' || strings.HasPrefix(s, "15") {
			return "", ErrTelefono
		}
		return s, nil
	case 12:
		// El 15 va después de la característica, que tiene 2 (11, Buenos Aires), 3 o 4 dígitos.
		for _, i := range []int{2, 3, 4} {
			if i == 2 && !strings.HasPrefix(s, "11") {
				continue
			}
			if s[i:i+2] == "15" {
				return nacional(s[:i] + s[i+2:])
			}
		}
	}
	return "", ErrTelefono
}
