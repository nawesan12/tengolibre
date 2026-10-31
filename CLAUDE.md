# TengoLibre

Turnos online para negocios de servicios (barberías, salones, estética). Es Jiku
(`~/Trabajo/Codigo/desarrollo-en-masa/jiku`, repo `nawesan12/aktivo`, en jikuapp.com) reescrito
dentro de Grupo Vimo. Jiku no se toca: es la referencia funcional (su `prisma/schema.prisma` es la
fuente del modelo) y su base no se migra; TengoLibre arranca con base nueva.

- `apps/api`: API en Go. Calca la forma de `~/Trabajo/vimo/mostrador-v2/apps/api`: stdlib mux,
  pgx, goose, RLS con `set_config('app.negocio_id')` vía `db.Pool.EnNegocio`, roles sin BYPASSRLS
  (`tengolibre_app`, `tengolibre_migrador`). pgxpool sólo en `internal/plataforma/db` (test).
- `apps/web`: todavía no existe. Se arma en SvelteKit con el handoff del rediseño, fiel al handoff.
- Login: Clerk (como todo Grupo Vimo).
- Producción: VPS compartido (ver `infra/vps/` y `docs/estado.md`). Corredor y Tu Local Web viven
  ahí también: no tocar sus archivos, y avisar a esas sesiones antes de tocar Caddy o Docker.

## Comandos

```bash
docker compose -f infra/compose.dev.yml up -d   # Postgres en 127.0.0.1:5446
cd apps/api
go run ./cmd/tengolibrectl bootstrap && go run ./cmd/tengolibrectl migrar
go run ./cmd/api                                # :4200
go vet ./... && go test ./...
```

Todo a main. Antes de una migración en producción: pg_dump.
