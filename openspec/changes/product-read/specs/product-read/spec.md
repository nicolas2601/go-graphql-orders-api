# Product read use case

## ADDED Requirements

### Requirement: Paginated product listing
El sistema SHALL devolver una pagina de productos que cumplen el filtro, junto con el total sin
paginar. SHALL acotar los parametros de paginacion: una pagina menor a uno se trata como la primera,
un tamano de pagina invalido usa el default, y un tamano mayor al maximo se recorta al maximo.

#### Scenario: Listing returns a page with total
- **WHEN** se listan productos con paginacion valida
- **THEN** se devuelven los items de esa pagina junto con el total sin paginar

#### Scenario: Pagination parameters are clamped
- **WHEN** se listan productos con page menor a uno o pageSize fuera de rango
- **THEN** el repositorio recibe una pagina y un tamano acotados a los limites permitidos

### Requirement: Product detail
El sistema SHALL devolver un producto por id, o ErrProductNotFound si no existe.

#### Scenario: Existing product
- **WHEN** se pide un producto que existe
- **THEN** se devuelve el producto

#### Scenario: Missing product
- **WHEN** se pide un producto que no existe
- **THEN** se devuelve ErrProductNotFound
