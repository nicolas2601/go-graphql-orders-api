# GraphQL delivery

## Why

Hay que exponer los casos de uso por GraphQL: registro/login/refresh, catalogo, ordenes. La capa de
delivery traduce entre GraphQL y los casos de uso, autentica por token, mapea errores a codigos
estables y aplica controles anti-DoS. No contiene logica de negocio.

## What Changes

- Esquema GraphQL final (con me, refreshToken, confirmOrder, hasNextPage, scalar Time) y codigo gqlgen.
- Resolvers que delegan en los casos de uso; Order.user y OrderItem.product se resuelven por su
  propio resolver (para el DataLoader del siguiente change).
- UserUseCase (lectura de usuario) para la query me y el resolver del dueno de la orden.
- Mapeo de errores tipados del dominio a extensions.code; los errores inesperados se enmascaran.
- Middleware de auth: Bearer -> identidad en el contexto; los resolvers autorizan por dueno.
- Handler HTTP con limite de complejidad, limite de tamano de body, e introspection/playground solo
  en desarrollo (fail-safe).

## Capabilities

- graphql-delivery

## Impact

- Nuevos: internal/delivery/graphql (resolvers, model, errors, mapping, authctx, middleware),
  internal/server, graph/schema.graphqls, gqlgen.yml. Deps: gqlgen, gqlparser.
