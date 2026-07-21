# Composition root

## ADDED Requirements

### Requirement: Startup wiring
El sistema SHALL, al arrancar, conectar a la base, aplicar migraciones, opcionalmente sembrar
productos, cablear dependencias y servir la API, y SHALL abortar si falta configuracion obligatoria
(DATABASE_URL) o si el secreto JWT es debil.

#### Scenario: Missing database URL
- **WHEN** se arranca sin DATABASE_URL
- **THEN** la app aborta con error en vez de servir

### Requirement: Readiness check
El sistema SHALL exponer /readyz que verifica la conectividad a la base, devolviendo 200 si esta
lista o 503 si no.

#### Scenario: Database reachable
- **WHEN** se consulta /readyz con la base disponible
- **THEN** responde 200

#### Scenario: Database unreachable
- **WHEN** se consulta /readyz y el check de la base falla
- **THEN** responde 503

### Requirement: Graceful shutdown
El sistema SHALL apagarse de forma ordenada ante SIGINT/SIGTERM, dejando de aceptar conexiones y
cerrando lo abierto dentro de un timeout.

#### Scenario: Shutdown signal
- **WHEN** el proceso recibe una senal de terminacion
- **THEN** el servidor deja de escuchar y retorna sin error

### Requirement: Auth rate limiting
El sistema SHALL limitar por IP la tasa de operaciones de autenticacion (login/register) para frenar
fuerza bruta.

#### Scenario: Too many auth attempts
- **WHEN** una IP supera el limite de intentos de auth
- **THEN** las operaciones de auth se rechazan con code RATE_LIMITED
