# DataLoader

## ADDED Requirements

### Requirement: Batched loading of related entities
El sistema SHALL agrupar las cargas de usuarios y productos de una misma request en una sola
consulta por tipo, en vez de una consulta por objeto (evita el N+1).

#### Scenario: Concurrent loads are batched
- **WHEN** varios resolvers cargan productos por id concurrentemente en una request
- **THEN** el repositorio recibe una sola consulta batcheada con todos los ids

### Requirement: Per-request loaders
El sistema SHALL crear loaders frescos por request, sin compartir cache entre requests de usuarios
distintos.

#### Scenario: Loaders isolated per request
- **WHEN** llega una request
- **THEN** se usa un juego de loaders propio de esa request
