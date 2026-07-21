# syntax=docker/dockerfile:1

# Etapa de build: compila un binario estatico y stripeado.
FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/orders-api ./cmd

# Etapa de runtime: imagen minima, usuario no-root, con healthcheck sobre /healthz.
# El wget del healthcheck lo provee busybox (ya incluido en la base alpine).
FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 1001 appuser
COPY --from=build /out/orders-api /usr/local/bin/orders-api
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["orders-api"]
