# Estado

## 31/10/2026 — boilerplate
- API en Go con `/salud/vivo`, `/salud/listo` y `/v1/version`; migración base con `negocios` y RLS.
- En el VPS: `/opt/tengolibre` (compose `tengolibre`: postgres + api en 127.0.0.1:4200),
  `/etc/caddy/sitios/tengolibre.caddy`, certificado de origen en `/etc/caddy/certs/tengolibre.*`.
  `tengolibre.com.ar` sirve una página de "próximamente" (`/opt/tengolibre/proximamente`).
- Puertos reservados: 4200 api, 3400 web.
- Copia diaria: la suma `vimo-copia` (la lleva la sesión de Tu Local Web).

## Desplegar la API
```bash
cd apps/api
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o bin/api ./cmd/api
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tengolibrectl ./cmd/tengolibrectl
scp bin/api deploy@187.41.102.190:/opt/tengolibre/bin/api.nuevo   # ídem tengolibrectl
ssh deploy@187.41.102.190 'cd /opt/tengolibre && mv bin/api.nuevo bin/api && docker compose restart api'
# migraciones:
ssh deploy@187.41.102.190 'cd /opt/tengolibre && docker compose run --rm --no-deps -T api /app/tengolibrectl migrar'
```

## Lo que sigue
1. Portar el modelo de Jiku por tandas (negocios y equipo → servicios y horarios → turnos →
   clientes → cobros con Mercado Pago → membresías, cupones, reseñas, notificaciones).
2. Ingreso con Clerk.
3. Front en SvelteKit con el handoff del rediseño (en 127.0.0.1:3400).
