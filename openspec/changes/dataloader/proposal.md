# DataLoader

## Why

Order.user y OrderItem.product se resolvian con una consulta por objeto (N+1). Con muchas ordenes o
muchas lineas, eso dispara decenas de consultas. DataLoader agrupa las cargas individuales de una
request en una sola consulta.

## What Changes

- Loaders por request (dataloadgen, NewMappedLoader sobre FindByIDs) para usuarios y productos.
- Middleware que inyecta loaders frescos por request en el contexto (sin cache compartida entre users).
- Los resolvers Order.user y OrderItem.product cargan por DataLoader en vez de por consulta directa.

## Capabilities

- dataloader

## Impact

- Nuevo internal/delivery/graphql/loaders. server.NewHandler recibe los repos para el middleware.
