// Package correo manda mails. Tres implementaciones: SMTP (el servidor de correo propio del VPS con
// usuario y TLS, o Mailpit en desarrollo), Log (desarrollo sin SMTP: el mail va al log) y Memoria
// (tests: guarda lo enviado).
package correo

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

// Adjunto de un mail (el .ics del turno).
type Adjunto struct {
	Nombre, Tipo string
	Datos        []byte
}

// Mensaje es un mail. HTML y Texto van los dos (los clientes que no muestran HTML usan el texto).
// ResponderA (opcional): a dónde va la respuesta, p. ej. el negocio en los mails al cliente.
// Baja (opcional): el link para dejar de recibirlos (va en List-Unsubscribe).
type Mensaje struct {
	Para, Asunto, HTML, Texto string
	ResponderA                string
	Baja                      string
	Adjuntos                  []Adjunto
}

// Correo manda un mensaje.
type Correo interface {
	Enviar(ctx context.Context, m Mensaje) error
}

// Elegir arma el correo según la configuración: SMTP si hay dirección; si no, el log.
func Elegir(direccion, usuario, clave, de string) Correo {
	if direccion == "" {
		return &Log{}
	}
	return &SMTP{Direccion: direccion, De: de, Usuario: usuario, Clave: clave}
}

// SMTP manda por un servidor SMTP. Sin Usuario, sin autenticación (Mailpit). Con Usuario y Clave,
// el servidor propio: en el puerto 465 con TLS desde el arranque y en los demás con STARTTLS.
type SMTP struct{ Direccion, De, Usuario, Clave string }

func (s *SMTP) Enviar(_ context.Context, m Mensaje) error {
	cuerpo, err := Armar(s.De, m, time.Now())
	if err != nil {
		return err
	}
	if s.Usuario == "" {
		return smtp.SendMail(s.Direccion, nil, direccionDe(s.De), []string{m.Para}, cuerpo)
	}
	return s.conUsuario(m.Para, cuerpo)
}

// Armar escribe el mail entero (encabezados y partes MIME). Exportado para los tests.
func Armar(remitente string, m Mensaje, ahora time.Time) ([]byte, error) {
	// Para, Asunto y los links van en encabezados: un salto de línea metería encabezados ajenos.
	if strings.ContainsAny(m.Para+m.Asunto+m.ResponderA+m.Baja, "\r\n") {
		return nil, fmt.Errorf("correo: salto de línea en un encabezado")
	}
	var b bytes.Buffer
	// El asunto va codificado (RFC 2047): con tildes en crudo, varios clientes lo muestran roto.
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\n",
		de(remitente), m.Para, mime.QEncoding.Encode("utf-8", m.Asunto), ahora.Format(time.RFC1123Z), idMensaje(remitente))
	if m.ResponderA != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", m.ResponderA)
	}
	if m.Baja != "" {
		fmt.Fprintf(&b, "List-Unsubscribe: <%s>\r\n", m.Baja)
	}
	alternativa := "tengolibre-alt-" + aleatorio()
	if len(m.Adjuntos) == 0 {
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", alternativa)
		escribirAlternativa(&b, alternativa, m)
		return b.Bytes(), nil
	}
	mixta := "tengolibre-mix-" + aleatorio()
	fmt.Fprintf(&b, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", mixta)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: multipart/alternative; boundary=%s\r\n\r\n", mixta, alternativa)
	escribirAlternativa(&b, alternativa, m)
	for _, a := range m.Adjuntos {
		if strings.ContainsAny(a.Nombre+a.Tipo, "\r\n\"") {
			return nil, fmt.Errorf("correo: nombre o tipo de adjunto inválido")
		}
		fmt.Fprintf(&b, "--%s\r\nContent-Type: %s; name=\"%s\"\r\nContent-Disposition: attachment; filename=\"%s\"\r\n"+
			"Content-Transfer-Encoding: base64\r\n\r\n", mixta, a.Tipo, a.Nombre, a.Nombre)
		enc := base64.StdEncoding.EncodeToString(a.Datos)
		for len(enc) > 76 {
			b.WriteString(enc[:76] + "\r\n")
			enc = enc[76:]
		}
		b.WriteString(enc + "\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", mixta)
	return b.Bytes(), nil
}

// escribirAlternativa: texto y HTML en quoted-printable (las líneas largas del HTML no rompen el
// límite de 998 caracteres de SMTP).
func escribirAlternativa(b *bytes.Buffer, frontera string, m Mensaje) {
	parte := func(tipo, contenido string) {
		fmt.Fprintf(b, "--%s\r\nContent-Type: %s; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", frontera, tipo)
		w := quotedprintable.NewWriter(b)
		_, _ = w.Write([]byte(contenido))
		_ = w.Close()
		b.WriteString("\r\n")
	}
	parte("text/plain", m.Texto)
	parte("text/html", m.HTML)
	fmt.Fprintf(b, "--%s--\r\n", frontera)
}

// conUsuario manda autenticado y siempre cifrado: nunca manda la clave en claro.
func (s *SMTP) conUsuario(para string, cuerpo []byte) error {
	host, puerto, err := net.SplitHostPort(s.Direccion)
	if err != nil {
		return fmt.Errorf("correo: dirección SMTP inválida: %w", err)
	}
	cfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	d := &net.Dialer{Timeout: 20 * time.Second}
	var c *smtp.Client
	if puerto == "465" {
		con, err := tls.DialWithDialer(d, "tcp", s.Direccion, cfg)
		if err != nil {
			return err
		}
		if c, err = smtp.NewClient(con, host); err != nil {
			return err
		}
	} else {
		con, err := d.Dial("tcp", s.Direccion)
		if err != nil {
			return err
		}
		if c, err = smtp.NewClient(con, host); err != nil {
			return err
		}
		if err := c.StartTLS(cfg); err != nil {
			return fmt.Errorf("correo: el servidor no acepta STARTTLS: %w", err)
		}
	}
	defer c.Close()
	if err := c.Auth(smtp.PlainAuth("", s.Usuario, s.Clave, host)); err != nil {
		return err
	}
	if err := c.Mail(direccionDe(s.De)); err != nil {
		return err
	}
	if err := c.Rcpt(para); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(cuerpo); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// de codifica el nombre del remitente si tiene caracteres fuera de ASCII.
func de(remitente string) string {
	i := strings.LastIndex(remitente, "<")
	if i <= 0 {
		return remitente
	}
	return mime.QEncoding.Encode("utf-8", strings.TrimSpace(remitente[:i])) + " " + remitente[i:]
}

// idMensaje arma un Message-ID único con el dominio del remitente (sin él, los filtros de spam restan).
func idMensaje(remitente string) string {
	dominio := "localhost"
	if d := direccionDe(remitente); strings.Contains(d, "@") {
		dominio = d[strings.LastIndex(d, "@")+1:]
	}
	return "<" + aleatorio() + "@" + dominio + ">"
}

func aleatorio() string {
	r := make([]byte, 12)
	_, _ = rand.Read(r)
	return hex.EncodeToString(r)
}

func direccionDe(de string) string {
	if i := strings.LastIndex(de, "<"); i >= 0 {
		return strings.TrimSuffix(de[i+1:], ">")
	}
	return de
}

// Log escribe el mail en el log en vez de mandarlo (desarrollo sin SMTP).
type Log struct{ Log *slog.Logger }

func (l *Log) Enviar(_ context.Context, m Mensaje) error {
	log := l.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("correo (sin SMTP, no sale)", "para", m.Para, "asunto", m.Asunto, "adjuntos", len(m.Adjuntos), "texto", m.Texto)
	return nil
}

// Memoria guarda los mensajes (tests). Con Fallar, devuelve ese error y no guarda nada.
type Memoria struct {
	mu       sync.Mutex
	Enviados []Mensaje
	Fallar   error
}

func (m *Memoria) Enviar(_ context.Context, msj Mensaje) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Fallar != nil {
		return m.Fallar
	}
	m.Enviados = append(m.Enviados, msj)
	return nil
}

// Para devuelve los mensajes mandados a esa dirección, en orden.
func (m *Memoria) Para(para string) []Mensaje {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Mensaje
	for _, x := range m.Enviados {
		if x.Para == para {
			out = append(out, x)
		}
	}
	return out
}

// Cantidad de mensajes mandados.
func (m *Memoria) Cantidad() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Enviados)
}
