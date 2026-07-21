# Authentication

## Why

La API expone operaciones protegidas que exigen un usuario autenticado. Se necesita registro y
login que emitan tokens JWT (access + refresh), y una rotacion de tokens. La autenticacion es
stateless: no hay sesion en el servidor.

## What Changes

- Caso de uso de autenticacion: register, login y refresh.
- Implementaciones de infraestructura de los ports del dominio: PasswordHasher con bcrypt y
  TokenService con JWT (access de vida corta, refresh de vida larga, con claim de tipo para que
  un token no sirva en el rol del otro).
- Los errores de credenciales no filtran si el fallo fue por email inexistente o contrasena
  incorrecta: ambos devuelven ErrInvalidCredentials.

## Capabilities

- authentication

## Impact

- Nuevo paquete internal/usecase (auth_usecase.go) e internal/auth (bcrypt, jwt).
- Dependencias: github.com/golang-jwt/jwt/v5, golang.org/x/crypto/bcrypt.
