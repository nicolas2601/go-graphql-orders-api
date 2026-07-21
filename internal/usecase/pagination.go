package usecase

const (
	// defaultPageSize es el tamano de pagina cuando no se pide uno valido.
	defaultPageSize = 20
	// maxPageSize acota el tamano de pagina para no permitir listados abusivos (vector de DoS).
	maxPageSize = 100
)

// Page es una pagina generica de resultados: los items mas los metadatos de paginacion.
type Page[T any] struct {
	Items    []T
	Total    int
	Page     int
	PageSize int
}

// TotalPages es la cantidad de paginas para el total y el tamano de pagina actuales.
func (p Page[T]) TotalPages() int {
	if p.PageSize <= 0 {
		return 0
	}
	return (p.Total + p.PageSize - 1) / p.PageSize
}

// HasNextPage indica si existe una pagina siguiente. Lo calcula el tipo generico una sola vez para
// que cada consumidor (por ejemplo los resolvers) no repita la aritmetica.
func (p Page[T]) HasNextPage() bool {
	return p.Page < p.TotalPages()
}

// normalizePagination acota page y pageSize a rangos validos. Se aplica en la capa de aplicacion
// para que ninguna implementacion de repositorio tenga que sanear estos valores por su cuenta.
func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch {
	case pageSize < 1:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}
	return page, pageSize
}
