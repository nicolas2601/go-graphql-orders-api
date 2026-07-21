# API de Órdenes - Go + GraphQL + PostgreSQL

API de gestión de órdenes de compra construida con Go, GraphQL (gqlgen) y PostgreSQL, siguiendo
Clean Architecture. Maneja usuarios, productos y órdenes con sus ítems, con autenticación stateless
por JWT y descuento de stock transaccional. Es mi solución a la Prueba 2 de la evaluación técnica.

## Stack

- Go 1.26
- GraphQL con [gqlgen](https://gqlgen.com/)
- PostgreSQL vía [pgx](https://github.com/jackc/pgx)
- Migraciones con [goose](https://github.com/pressly/goose) (embebidas en el binario)
- JWT con [golang-jwt](https://github.com/golang-jwt/jwt) (access + refresh)
- DataLoader con [dataloadgen](https://github.com/vikstrous/dataloadgen) para resolver el N+1
- Tests de integración con [testcontainers-go](https://testcontainers.com/)
- Docker + docker-compose, CI con GitHub Actions

## Prerrequisitos

- Docker (para `docker compose` y para los tests de integración con testcontainers)
- Go >= 1.26 (solo si se corre localmente sin Docker)
- `golangci-lint` v2 (solo para `make lint` local)

## Cómo ejecutarlo

### Con Docker Compose (recomendado)

Levanta la API junto a PostgreSQL, todo en contenedores. La API espera a que la base esté healthy,
aplica las migraciones y siembra productos iniciales al arrancar:

```bash
docker compose up --build
```

- **GraphQL**: `POST http://localhost:8080/query`
- **Playground** (solo en `development`): `http://localhost:8080/`
- **Liveness**: `GET http://localhost:8080/healthz`
- **Readiness** (verifica la base): `GET http://localhost:8080/readyz`

El puerto de la API es configurable con `API_PORT`. Para un despliegue tipo producción se cambia
`APP_ENV=production` (apaga el playground y la introspection) y se setea un `JWT_SECRET` real.

### Localmente (contra un PostgreSQL propio)

```bash
DATABASE_URL="postgres://orders:orders@localhost:5432/orders?sslmode=disable" \
  JWT_SECRET="una-clave-de-al-menos-16-bytes" APP_ENV=development make run
```

La app corre las migraciones y el seed al arrancar; no requiere pasos previos.

## Variables de entorno

| Variable | Default | Uso |
|---|---|---|
| `PORT` | `8080` | Puerto del servidor HTTP |
| `APP_ENV` | `production` | `development` habilita playground e introspection; `production` es el default fail-safe |
| `LOG_LEVEL` | `info` | Nivel de logging estructurado (`debug`/`info`/`warn`/`error`) |
| `DATABASE_URL` | vacío | Cadena de conexión a PostgreSQL (obligatoria) |
| `JWT_SECRET` | vacío | Secreto de firma HS256; obligatorio y >= 16 bytes, si no la app no arranca |
| `JWT_ACCESS_TTL` | `15m` | Vida del access token |
| `JWT_REFRESH_TTL` | `168h` | Vida del refresh token |
| `BCRYPT_COST` | `10` | Cost de bcrypt para el hash de contraseñas |
| `SEED_PRODUCTS` | `true` | Sembrar productos iniciales al arrancar |
| `AUTH_RATE_LIMIT` | `10` | Intentos de login/register por minuto por IP |

`.env.example` documenta los valores; el `.env` real no se versiona. La app lee su configuración del
entorno; el `.env` solo lo usa `docker compose` para interpolar variables.

## Decisión de diseño: manejo de transacciones

La creación de una orden valida y descuenta el stock de cada producto y persiste la orden **en una
sola transacción**, sin exponer el `*sql.Tx` al caso de uso. Se resuelve con un **Transaction Manager
que propaga la transacción por el `context`**: el dominio define la interfaz `TxManager.WithinTx(ctx,
fn)`; su implementación PostgreSQL abre la transacción, la inyecta en el contexto y hace commit o
rollback según el resultado de `fn`. Los repositorios resuelven su ejecutor con un helper
`querier(ctx)` que devuelve la transacción del contexto o el pool, de modo que son agnósticos a si
corren dentro de una transacción. El descuento de stock es atómico (`UPDATE ... WHERE stock >= qty`),
lo que evita sobreventa bajo concurrencia sin locks explícitos. Se descartó la alternativa de un
Unit of Work explícito (más boilerplate) y la de meter la lógica en el repositorio (rompería la
separación de capas).

## Arquitectura (Clean Architecture)

Las dependencias apuntan hacia adentro: el dominio no conoce infraestructura, y los casos de uso
dependen de interfaces definidas en el dominio, no de implementaciones concretas.

```
cmd/main.go                  composition root: config, DB, migraciones, seed, DI, servidor
        |
        v
internal/server              handler HTTP: request-id, recover, auth, DataLoaders, anti-DoS, health
        |
        v
internal/delivery/graphql    resolvers (SIN lógica) + mapeo de errores + DataLoaders + middleware
        |
        v
internal/usecase             casos de uso: auth, product, order (orquestan la transacción)
        |
        v
internal/domain              entidades, invariantes, errores tipados, INTERFACES (repos, tx, tokens)
        ^
        |
internal/repository/postgres · internal/auth   implementaciones de las interfaces del dominio
```

## API GraphQL

**Queries**
- `me`: usuario autenticado (o null).
- `products(filter, page, pageSize)`: catálogo paginado, con filtro por nombre y rango de precio.
- `product(id)`: detalle de un producto.
- `myOrders(page, pageSize)`: órdenes del usuario autenticado.
- `order(id)`: detalle de una orden (solo del dueño).

**Mutations**
- `register(input)` / `login(input)`: devuelven `AuthPayload { accessToken, refreshToken, user }`.
- `refreshToken(token)`: rota el par de tokens.
- `createOrder(input)`: crea una orden descontando stock atómicamente.
- `confirmOrder(id)` / `cancelOrder(id)`: transiciones de estado (cancelar restaura el stock).

Las operaciones (salvo register/login/refreshToken) requieren el header `Authorization: Bearer
<accessToken>`.

Ejemplo de registro:

```graphql
mutation {
  register(input: { email: "nico@example.com", password: "secret123" }) {
    accessToken
    user { id email }
  }
}
```

Ejemplo de creación de orden (con el token en el header `Authorization`):

```graphql
mutation {
  createOrder(input: { items: [{ productId: "…", quantity: 2 }] }) {
    id
    total
    status
    items { quantity unitPrice product { name } }
  }
}
```

Los errores de dominio se mapean a un código estable en `extensions.code` (por ejemplo
`INSUFFICIENT_STOCK`, `ORDER_NOT_OWNED`, `UNAUTHENTICATED`); los errores inesperados se enmascaran
como `INTERNAL_ERROR` sin filtrar detalles.

## Autenticación

Stateless por JWT. `login`/`register` emiten un access token corto y un refresh token largo, con un
claim de tipo para que uno no sirva en el rol del otro. El middleware de auth adjunta la identidad al
contexto a partir del access token; los resolvers autorizan por dueño (una orden solo la ve/cancela/
confirma su dueño). El servicio de tokens vive detrás de una interfaz del dominio, no hardcodeado en
los resolvers.

## Tests

```bash
make test   # go test ./... -race -covermode=atomic -coverprofile=coverage.out
```

La lógica de negocio (dominio, casos de uso) está cubierta con tests unitarios contra dobles; la
persistencia y el flujo completo se prueban con **testcontainers** contra un PostgreSQL real (por lo
que requieren Docker). Entre las garantías verificadas de punta a punta: la transacción de una orden
se revierte entera si un producto posterior no tiene stock; 20 órdenes concurrentes sobre 5 unidades
de stock tienen éxito exactamente 5 veces (sin sobreventa, bajo `-race`); el DataLoader agrupa cargas
concurrentes en una sola consulta; y la autorización por dueño se cumple end-to-end por GraphQL.

## Integración continua

El pipeline de GitHub Actions (`.github/workflows/ci.yml`) corre en cada push y pull request a
`main` y `develop`: lint (`golangci-lint` con `gosec`), build, verificación de `go mod tidy`, tests
con race detector (incluye los de PostgreSQL vía testcontainers), `govulncheck` (bloqueante) y build
de la imagen Docker.

## Estructura del proyecto

```
cmd/main.go                     composition root
internal/
  config/                       configuración desde el entorno
  domain/                       entidades, invariantes, errores, interfaces (ports)
  usecase/                      casos de uso (auth, product, order) + paginación
  auth/                         bcrypt + JWT (implementaciones de las interfaces del dominio)
  repository/postgres/          repos pgx, TxManager, migraciones, seed
  delivery/graphql/             resolvers, mapeo, DataLoaders, middleware, rate limit
  server/                       handler HTTP
migrations/                     migraciones goose (embebidas)
graph/schema.graphqls           esquema GraphQL
openspec/specs/                 especificaciones por capacidad (framework OpenSpec)
Dockerfile, docker-compose.yml  empaquetado y orquestación
.github/workflows/ci.yml        pipeline de CI
```

## Decisiones de diseño y deuda conocida

- **`price`/`total`/`unit_price` como `float64` / `DOUBLE PRECISION`**: heredado del tipo `Float` del
  esquema GraphQL. Para dinero real lo correcto sería `NUMERIC`/decimal en centavos; deuda consciente.
- **`unitPrice` como snapshot**: se congela el precio del producto al crear la orden, para que un
  cambio de precio posterior no altere las órdenes históricas.
- **`createdAt` como scalar `Time`** (RFC3339 UTC): mejora sobre el `String` del enunciado, para
  normalizar formato y zona horaria.
- **`ORDER_NOT_OWNED` distinguible de `ORDER_NOT_FOUND`**: el enunciado pide mapear `ErrOrderNotOwned`
  como error tipado, así que se expone. Trade-off: revela existencia de órdenes ajenas, mitigado
  porque los ids son UUID no enumerables.
- **Refresh token stateless con rotación**: sin tabla de revocación; para logout/revocación real se
  persistiría el hash del refresh con rotación.
- **Rate limiter por IP**: buckets en memoria sin TTL; a gran escala se evolucionaría a una cache con
  expiración.
- **Imagen Docker sobre Alpine** (no distroless) por la simplicidad del healthcheck; sin pinning por
  digest ni límites de recursos. Para producción endurecida, distroless + digest + límites.
- **CORS no configurado**: el enunciado indica "solo el servidor" (sin cliente browser); se agregaría
  con un frontend.
- **`/healthz` es liveness; `/readyz` es readiness** (hace `pool.Ping`).
```
