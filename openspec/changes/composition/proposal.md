# Composition root

## Why

Falta el punto de entrada que ensambla todo y hace la app ejecutable: cargar config, conectar a
Postgres, correr migraciones y seed, cablear dependencias, y servir con apagado ordenado. Ademas se
suman controles operativos y de robustez que son responsabilidad de la capa de composicion.

## What Changes

- cmd/main.go: config -> pool pgx -> migraciones -> seed -> DI -> servidor con graceful shutdown.
- Seed idempotente de productos iniciales.
- /readyz con ping a la base (readiness real); /healthz sigue siendo liveness.
- Middlewares: request-id (correlacion), recover (un panic no tumba el proceso), client-ip.
- Rate limiting por IP en las mutations de auth (login/register) para frenar fuerza bruta.
- El secreto JWT se valida al construir el servicio: si es debil, la app no arranca.

## Capabilities

- composition

## Impact

- Nuevos: cmd/main.go (completo), internal/repository/postgres/seed.go,
  internal/delivery/graphql/ratelimit, middlewares de observabilidad. server.NewHandler recibe Deps.
