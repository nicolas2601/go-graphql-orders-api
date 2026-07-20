# API de Órdenes · Go + GraphQL + PostgreSQL

Esta es mi solución a la **Prueba 2** de la evaluación técnica: una API de gestión de
órdenes de compra. El sistema maneja usuarios, productos y órdenes con sus ítems. Se evalúa
diseñar un dominio con relaciones, manejar transacciones y autenticación stateless.

## Stack

- **Go**
- **GraphQL** con [gqlgen](https://gqlgen.com/)
- **PostgreSQL** (vía `pgx` o `sqlx`)
- **JWT** para autenticación stateless
- **DataLoader** para resolver el problema N+1

## Arquitectura

**Clean Architecture** con 4 capas: dominio, casos de uso, adaptadores/repositorio y
delivery. Puntos clave del diseño:

- Las interfaces del repositorio viven en la capa de dominio.
- El servicio JWT está detrás de una interfaz (no hardcodeado en los resolvers).
- La creación de una orden y el descuento de stock ocurren en **una sola transacción**; la
  interfaz del repositorio soporta esto **sin exponer `*sql.Tx`** al caso de uso.
- Errores de dominio tipados (`ErrProductNotFound`, `ErrInsufficientStock`,
  `ErrOrderNotOwned`, …) mapeados a errores GraphQL con `code` y `message`.

```
cmd/
internal/
  domain/                   entidades + errores tipados + interfaces del repositorio
  usecase/                  casos de uso (auth, product, order)
  repository/postgres/      implementación PostgreSQL
  delivery/graphql/         resolvers + dataloader + middleware de auth
migrations/                 migraciones (goose o migrate)
graph/schema.graphqls
docker-compose.yml          levanta la app + Postgres
```

## Cómo correrlo

> 🚧 Las instrucciones de arranque (Docker Compose + variables de entorno) se completan al
> terminar la implementación.

## Estado

En construcción — el historial de commits refleja el progreso incremental.
