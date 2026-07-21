# Deployment

## Why

La app necesita empaquetarse y levantarse de forma reproducible. Se entrega una imagen Docker
minima y no-root, y un docker-compose que levanta la API junto a PostgreSQL.

## What Changes

- Dockerfile multi-stage: build del binario estatico y stripeado, runtime alpine con usuario no-root
  y healthcheck sobre /healthz.
- docker-compose.yml: servicio api + servicio db (postgres:16-alpine) con healthcheck y depends_on;
  la base no publica puerto al host (solo red interna).
- .dockerignore para un contexto de build minimo.
- Job de Docker en CI que construye la imagen.

## Capabilities

- deployment

## Impact

- Nuevos: Dockerfile, docker-compose.yml, .dockerignore. Job docker en ci.yml.
