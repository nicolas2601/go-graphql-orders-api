# Order management use case

## Why

Es el nucleo de la API: crear ordenes descontando stock de forma atomica, listar las ordenes del
usuario, ver el detalle (solo del dueno), cancelar (restaurando stock) y confirmar. La logica de
negocio vive en el caso de uso; la transaccion se maneja detras del port TxManager sin exponer SQL.

## What Changes

- OrderUseCase con: Create, MyOrders, Get, Cancel, Confirm.
- Create y Cancel corren dentro de una transaccion (TxManager.WithinTx): Create valida y descuenta
  stock por producto y persiste la orden; Cancel restaura stock y cambia el estado. Todo o nada.
- Snapshot de unitPrice desde el precio actual del producto al crear la orden.
- Merge de lineas con el mismo producto antes de crear la orden (una sola linea por producto).
- Autorizacion por dueno: Get/Cancel/Confirm exigen que la orden pertenezca al usuario (ErrOrderNotOwned).

## Capabilities

- order-management

## Impact

- Nuevo internal/usecase/order_usecase.go. Reutiliza Page[T] y normalizePagination.
