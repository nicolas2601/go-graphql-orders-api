# GraphQL delivery

## ADDED Requirements

### Requirement: Authentication by token
El sistema SHALL adjuntar la identidad del usuario al contexto a partir de un access token valido en
el header Authorization. Las operaciones protegidas SHALL devolver UNAUTHENTICATED sin un usuario
autenticado; register, login y refresh SHALL ser publicas.

#### Scenario: Protected operation without a token
- **WHEN** se invoca createOrder sin token
- **THEN** se devuelve un error con code UNAUTHENTICATED

#### Scenario: Public operation without a token
- **WHEN** se invoca register sin token con datos validos
- **THEN** se crea el usuario y se devuelven tokens

### Requirement: Domain error mapping
El sistema SHALL mapear los errores tipados del dominio a un code estable en extensions, y SHALL
enmascarar los errores inesperados como error interno sin filtrar su mensaje.

#### Scenario: Known domain error
- **WHEN** un caso de uso devuelve un error tipado (por ejemplo stock insuficiente)
- **THEN** la respuesta incluye el code correspondiente (INSUFFICIENT_STOCK)

#### Scenario: Unexpected error
- **WHEN** un caso de uso devuelve un error no reconocido
- **THEN** la respuesta usa code INTERNAL_ERROR y un mensaje generico

### Requirement: Ownership on order operations
El sistema SHALL permitir ver, cancelar o confirmar una orden solo a su dueno, devolviendo
ORDER_NOT_OWNED en otro caso.

#### Scenario: Non-owner reads an order
- **WHEN** un usuario pide una orden que no le pertenece
- **THEN** se devuelve un error con code ORDER_NOT_OWNED

### Requirement: Anti-DoS controls
El sistema SHALL acotar el tamano del body y la complejidad de las queries, y SHALL habilitar
introspection y playground solo en modo desarrollo.

#### Scenario: Introspection in production
- **WHEN** se envia una query de introspection en produccion
- **THEN** la query es rechazada

#### Scenario: Query above the complexity limit
- **WHEN** se envia una query cuya complejidad supera el limite
- **THEN** la query es rechazada con un error de complejidad
