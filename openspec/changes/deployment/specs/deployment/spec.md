# Deployment

## ADDED Requirements

### Requirement: Container image
El sistema SHALL empaquetarse en una imagen Docker que corra como usuario no-root, con el binario
compilado en una etapa separada y un healthcheck sobre el endpoint de liveness.

#### Scenario: Image runs as non-root
- **WHEN** se construye y corre la imagen
- **THEN** el proceso corre como un usuario sin privilegios y expone /healthz

### Requirement: Compose stack
El sistema SHALL proveer un docker-compose que levante la API y PostgreSQL, esperando a que la base
este healthy antes de arrancar la API.

#### Scenario: Stack starts and serves
- **WHEN** se levanta el stack con docker compose
- **THEN** la API arranca tras la base, aplica migraciones y seed, y responde a las queries
