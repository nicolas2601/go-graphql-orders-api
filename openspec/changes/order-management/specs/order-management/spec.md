# Order management use case

## ADDED Requirements

### Requirement: Create order with atomic stock deduction
El sistema SHALL crear una orden validando y descontando el stock de cada producto dentro de una
sola transaccion, tomando el precio unitario del producto al momento de crear la orden. SHALL
fusionar las lineas con el mismo producto en una sola. Si algun producto no existe o no tiene stock
suficiente, SHALL abortar la transaccion sin persistir la orden ni descontar stock.

#### Scenario: Successful order deducts stock and snapshots price
- **WHEN** se crea una orden con productos con stock suficiente
- **THEN** se persiste la orden en estado PENDING, el total es la suma de cantidad por precio unitario, y el stock de cada producto queda descontado

#### Scenario: Insufficient stock aborts the order
- **WHEN** se crea una orden y algun producto no tiene stock suficiente
- **THEN** se devuelve ErrInsufficientStock y no se persiste la orden

#### Scenario: Duplicate product lines are merged
- **WHEN** se crea una orden con dos lineas del mismo producto
- **THEN** la orden tiene una sola linea para ese producto con la cantidad sumada

### Requirement: List own orders
El sistema SHALL devolver una pagina de las ordenes del usuario autenticado.

#### Scenario: User lists their orders
- **WHEN** un usuario lista sus ordenes
- **THEN** se devuelve una pagina con sus ordenes y el total

### Requirement: Ownership on order access
El sistema SHALL permitir ver, cancelar o confirmar una orden solo a su dueno; en otro caso SHALL
devolver ErrOrderNotOwned.

#### Scenario: Owner reads the order
- **WHEN** el dueno pide el detalle de su orden
- **THEN** se devuelve la orden

#### Scenario: Non-owner is rejected
- **WHEN** un usuario pide una orden que no le pertenece
- **THEN** se devuelve ErrOrderNotOwned

### Requirement: Cancel restores stock
El sistema SHALL cancelar una orden en estado PENDING restaurando el stock de sus productos dentro
de una transaccion, y SHALL rechazar la cancelacion de una orden que no este en PENDING.

#### Scenario: Cancel a pending order restores stock
- **WHEN** el dueno cancela una orden en estado PENDING
- **THEN** la orden pasa a CANCELLED y el stock de cada producto se restaura

#### Scenario: Cannot cancel a non-pending order
- **WHEN** se cancela una orden que no esta en PENDING
- **THEN** se devuelve ErrOrderNotPending

### Requirement: Confirm a pending order
El sistema SHALL confirmar una orden en estado PENDING del usuario; en otro estado SHALL devolver
ErrOrderNotPending.

#### Scenario: Confirm a pending order
- **WHEN** el dueno confirma una orden en estado PENDING
- **THEN** la orden pasa a CONFIRMED
