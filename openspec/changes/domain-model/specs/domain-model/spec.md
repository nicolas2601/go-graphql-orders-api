# Domain model

## ADDED Requirements

### Requirement: Product invariants
El sistema SHALL rechazar un producto cuyo nombre este vacio, cuyo precio no sea mayor a cero, o
cuyo stock sea negativo.

#### Scenario: Valid product
- **WHEN** se construye un producto con nombre no vacio, precio mayor a cero y stock no negativo
- **THEN** el producto se crea sin error

#### Scenario: Non-positive price
- **WHEN** se construye un producto con precio cero o negativo
- **THEN** se devuelve ErrInvalidPrice

#### Scenario: Negative stock
- **WHEN** se construye un producto con stock negativo
- **THEN** se devuelve ErrInvalidStock

### Requirement: Order creation invariants
El sistema SHALL crear una orden solo si tiene al menos un item, cada item con cantidad mayor a cero
y precio unitario no negativo, y SHALL calcular el total como la suma de cantidad por precio unitario.
Una orden recien creada SHALL quedar en estado PENDING.

#### Scenario: Order with items computes total
- **WHEN** se crea una orden con items validos
- **THEN** el total es la suma de cantidad por precio unitario de cada item y el estado es PENDING

#### Scenario: Empty order is rejected
- **WHEN** se crea una orden sin items
- **THEN** se devuelve ErrEmptyOrder

### Requirement: Order state transitions
El sistema SHALL permitir confirmar o cancelar una orden solo si esta en estado PENDING. Cancelar una
orden SHALL registrar el momento de cancelacion.

#### Scenario: Confirm a pending order
- **WHEN** se confirma una orden en estado PENDING
- **THEN** la orden pasa a estado CONFIRMED

#### Scenario: Cancel a pending order
- **WHEN** se cancela una orden en estado PENDING
- **THEN** la orden pasa a estado CANCELLED y se registra el momento de cancelacion

#### Scenario: Transition from a non-pending order is rejected
- **WHEN** se intenta confirmar o cancelar una orden que no esta en PENDING
- **THEN** se devuelve ErrOrderNotPending

### Requirement: Credential validation
El sistema SHALL validar que el email tenga un formato valido y que la contrasena tenga al menos
ocho caracteres e incluya al menos un numero.

#### Scenario: Invalid email format
- **WHEN** se valida un email sin formato valido
- **THEN** se devuelve ErrInvalidEmail

#### Scenario: Weak password
- **WHEN** se valida una contrasena con menos de ocho caracteres o sin ningun numero
- **THEN** se devuelve ErrInvalidPassword
