package trevol

import (
	"context"
	"iter"
)

// listPage es lo que tienen en común los sobres de lista generados (InvoiceList, EventList, …).
type listPage[T any] interface {
	GetData() []T
	GetHasMore() bool
	GetNextCursor() string
}

// Paginate recorre todas las páginas de un listado, perezosamente (iter.Seq2: for item, err := range …).
//
// page(cursor) pide UNA página (`starting_after = cursor`; nil la primera) y devuelve el sobre de lista. Sigue con next_cursor mientras
// has_more; si la página no lo trae, usa el id del último elemento (así lo arma la API). Si page devuelve un error, se entrega como último
// par (zero, err) y termina. Se detiene si el cursor no avanza: una API que repitiera el mismo cursor no puede dejar el bucle girando
// para siempre. Cancelar ctx también lo detiene.
//
//	for f, err := range trevol.Paginate(ctx, func(cursor *string) (*generado.InvoiceList, error) {
//		r := c.InvoicesAPI.ListInvoices(ctx).Limit(100)
//		if cursor != nil { r = r.StartingAfter(*cursor) }
//		l, _, err := r.Execute()
//		return l, err
//	}) { ... }
func Paginate[T any, L listPage[T]](ctx context.Context, page func(cursor *string) (L, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var cursor *string
		for {
			if err := ctx.Err(); err != nil {
				var cero T
				yield(cero, err)
				return
			}
			p, err := page(cursor)
			if err != nil {
				var cero T
				yield(cero, err)
				return
			}
			datos := p.GetData()
			for _, d := range datos {
				if !yield(d, nil) {
					return
				}
			}
			if !p.GetHasMore() || len(datos) == 0 {
				return
			}
			siguiente := p.GetNextCursor()
			if siguiente == "" {
				if g, ok := any(&datos[len(datos)-1]).(interface{ GetId() string }); ok {
					siguiente = g.GetId()
				}
			}
			if siguiente == "" || (cursor != nil && siguiente == *cursor) {
				return
			}
			cursor = &siguiente
		}
	}
}
