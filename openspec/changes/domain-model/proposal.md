# Domain model

## Why

La API de ordenes necesita un nucleo de dominio independiente de la infraestructura: entidades
con sus invariantes, errores tipados y las interfaces (ports) que las capas externas
implementaran. Es la base sobre la que se apoyan los casos de uso, la persistencia y la entrega.

## What Changes

- Entidades del dominio: User, Product, Order y OrderItem, con constructores que fuerzan sus invariantes.
- Reglas de dominio puras: validacion de email y de politica de contrasena; calculo del total de una
  orden; transiciones de estado (PENDING -> CONFIRMED, PENDING -> CANCELLED).
- Errores tipados (sentinels) para los resultados de negocio esperados.
- Interfaces (ports) en el dominio: repositorios de User/Product/Order, TxManager, TokenService y
  PasswordHasher. La logica de negocio dependera de estas interfaces, nunca de implementaciones concretas.

## Capabilities

- domain-model

## Impact

- Nuevos paquetes: internal/domain (entidades, errores, ports).
- Sin dependencias de infraestructura; base para usecase, repository y delivery.
