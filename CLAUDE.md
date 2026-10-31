# TengoLibre

Turnos online para negocios de servicios (barberías, salones, estética). Es Jiku
(`~/Trabajo/Codigo/desarrollo-en-masa/jiku`, repo `nawesan12/aktivo`, en jikuapp.com) reescrito
dentro de Grupo Vimo. Jiku no se toca: es la referencia funcional (su `prisma/schema.prisma` es la
fuente del modelo) y su base no se migra; TengoLibre arranca con base nueva.

- `apps/api`: API en Go. Calca la forma de `~/Trabajo/vimo/mostrador-v2/apps/api`: stdlib mux,
  pgx, goose, RLS con `set_config('app.negocio_id')` vía `db.Pool.EnNegocio`, roles sin BYPASSRLS
  (`tengolibre_app`, `tengolibre_migrador`). pgxpool sólo en `internal/plataforma/db` (test).
- `apps/web`: todavía no existe. Decidido (31/10): SvelteKit + adapter-node en 127.0.0.1:3400, calcando `mostrador-v2/apps/web` (ingreso con Clerk, cliente de la API) y la UI fiel al handoff del rediseño.
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

## Front (`apps/web`)

```bash
cd apps/web && bun install
bun run dev                     # con TENGOLIBRE_API=http://127.0.0.1:4200 y TENGOLIBRE_INGRESO_PRUEBA=1
bun run check && bun run test   # svelte-check y vitest
bunx playwright test            # e2e contra la API real (la levanta sola en :4290 y el front en :5193)
```

SvelteKit 3 + Svelte 5 (runes) + adapter-node. El navegador nunca habla con la API: lo hace el
servidor de SvelteKit (`#lib/server/api.ts`) con la sesión propia en la cookie httpOnly y el negocio
elegido en otra. Formularios con acciones y `intentar()` (`#lib/server/acciones.ts`). Tiempo real:
`/panel/eventos` trae el SSE de la API y `escucharPanel()` invalida `depends('panel:<tipo>')`.
Estilo provisorio y neutro en `app.css` (variables): la UI final sale del handoff.
