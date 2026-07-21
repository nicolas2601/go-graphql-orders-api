# Authentication

## ADDED Requirements

### Requirement: User registration
El sistema SHALL registrar un usuario nuevo validando email y contrasena, hasheando la contrasena,
y emitiendo un par de tokens (access y refresh). SHALL rechazar un email ya registrado.

#### Scenario: Successful registration
- **WHEN** se registra un email valido no usado con una contrasena valida
- **THEN** el usuario se persiste con la contrasena hasheada y se devuelve un par de tokens

#### Scenario: Duplicate email
- **WHEN** se registra un email que ya existe
- **THEN** se devuelve ErrEmailAlreadyRegistered y no se emite ningun token

#### Scenario: Invalid password
- **WHEN** se registra con una contrasena que no cumple la politica
- **THEN** se devuelve ErrInvalidPassword y no se persiste el usuario

### Requirement: User login
El sistema SHALL autenticar por email y contrasena y emitir un par de tokens. Un email inexistente
o una contrasena incorrecta SHALL devolver ErrInvalidCredentials sin distinguir cual fallo.

#### Scenario: Successful login
- **WHEN** las credenciales coinciden con un usuario existente
- **THEN** se devuelve un par de tokens

#### Scenario: Wrong password
- **WHEN** el email existe pero la contrasena no coincide
- **THEN** se devuelve ErrInvalidCredentials

#### Scenario: Unknown email
- **WHEN** el email no existe
- **THEN** se devuelve ErrInvalidCredentials

### Requirement: Token refresh
El sistema SHALL emitir un par de tokens nuevo a partir de un refresh token valido, y SHALL rechazar
un token invalido, expirado o de tipo incorrecto.

#### Scenario: Valid refresh token
- **WHEN** se presenta un refresh token valido de un usuario existente
- **THEN** se emite un par de tokens nuevo

#### Scenario: Access token used as refresh
- **WHEN** se presenta un access token en lugar de un refresh token
- **THEN** se devuelve ErrUnauthenticated
