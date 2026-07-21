# Product read use case

## Why

Los usuarios autenticados consultan el catalogo de productos: un listado paginado con filtro por
nombre y rango de precio, y el detalle de un producto. Los productos son de solo lectura para los
usuarios normales (no hay mutations de producto en la API); se cargan por seed.

## What Changes

- Caso de uso de productos: List (filtro + paginacion) y Get.
- Normalizacion de paginacion en la capa de aplicacion: page y pageSize se acotan (defaults y maximo)
  para que ninguna implementacion de repositorio reciba valores abusivos.
- Tipo de pagina generico reutilizable (Page[T]) para envolver items + total + page + pageSize.

## Capabilities

- product-read

## Impact

- Nuevo internal/usecase/product_usecase.go y pagination.go (helper compartido).
