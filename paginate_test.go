package trevol

import (
	"context"
	"errors"
	"testing"

	"github.com/trevol/trevol-go/generado"
)

func doc(n string) generado.Document {
	d := generado.NewDocument()
	d.SetId("inv_" + n)
	return *d
}

func pagina(has bool, cursor string, ids ...string) *generado.InvoiceList {
	l := generado.NewInvoiceList()
	for _, id := range ids {
		l.Data = append(l.Data, doc(id))
	}
	l.SetHasMore(has)
	if cursor != "" {
		l.SetNextCursor(cursor)
	}
	return l
}

func TestRecorreTresPaginasFalsasConNextCursorYSeDetiene(t *testing.T) {
	var pedidos []string
	paginas := []*generado.InvoiceList{pagina(true, "inv_2", "1", "2"), pagina(true, "inv_4", "3", "4"), pagina(false, "", "5")}
	var ids []string
	for f, err := range Paginate(context.Background(), func(c *string) (*generado.InvoiceList, error) {
		if c == nil {
			pedidos = append(pedidos, "")
		} else {
			pedidos = append(pedidos, *c)
		}
		return paginas[len(pedidos)-1], nil
	}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, f.GetId())
	}
	if len(ids) != 5 || ids[0] != "inv_1" || ids[4] != "inv_5" {
		t.Errorf("ids = %v", ids)
	}
	if len(pedidos) != 3 || pedidos[0] != "" || pedidos[1] != "inv_2" || pedidos[2] != "inv_4" {
		t.Errorf("pedidos = %v", pedidos)
	}
}

func TestSiHasMoreVieneSinNextCursorUsaElIDDelUltimo(t *testing.T) {
	var pedidos []string
	paginas := []*generado.InvoiceList{pagina(true, "", "1", "2"), pagina(false, "", "3")}
	n := 0
	for range Paginate(context.Background(), func(c *string) (*generado.InvoiceList, error) {
		if c != nil {
			pedidos = append(pedidos, *c)
		}
		n++
		return paginas[n-1], nil
	}) {
	}
	if n != 2 || len(pedidos) != 1 || pedidos[0] != "inv_2" {
		t.Errorf("n=%d pedidos=%v", n, pedidos)
	}
}

func TestEsPerezosa(t *testing.T) {
	llamadas := 0
	for range Paginate(context.Background(), func(c *string) (*generado.InvoiceList, error) {
		llamadas++
		return pagina(true, "inv_2", "1", "2"), nil
	}) {
		break
	}
	if llamadas != 1 {
		t.Errorf("llamadas = %d", llamadas)
	}
}

func TestUnCursorQueNoAvanzaOUnaPaginaVaciaNoDejanElBucleGirando(t *testing.T) {
	n := 0
	cuenta := 0
	for range Paginate(context.Background(), func(c *string) (*generado.InvoiceList, error) {
		n++
		return pagina(true, "inv_1", "1"), nil
	}) {
		cuenta++
	}
	if cuenta > 3 {
		t.Errorf("giró %d veces", cuenta)
	}
	for range Paginate(context.Background(), func(c *string) (*generado.InvoiceList, error) { return pagina(true, "x"), nil }) {
		t.Error("una página vacía no entrega nada")
	}
}

func TestUnErrorDeLaPaginaSeEntregaYTermina(t *testing.T) {
	boom := errors.New("boom")
	var errs []error
	for _, err := range Paginate(context.Background(), func(c *string) (*generado.InvoiceList, error) { return nil, boom }) {
		errs = append(errs, err)
	}
	if len(errs) != 1 || !errors.Is(errs[0], boom) {
		t.Errorf("%v", errs)
	}
}

func TestClientPaginateConUnMetodoGeneradoYElServidorFalso(t *testing.T) {
	b := nuevoBanco(t,
		jsonResp(200, map[string]any{"object": "list", "data": []any{map[string]any{"id": "inv_1"}, map[string]any{"id": "inv_2"}}, "has_more": true, "next_cursor": "inv_2"}),
		jsonResp(200, map[string]any{"object": "list", "data": []any{map[string]any{"id": "inv_3"}}, "has_more": false, "next_cursor": nil}))
	ctx := context.Background()
	var ids []string
	for f, err := range Paginate(ctx, func(cursor *string) (*generado.InvoiceList, error) {
		r := b.cliente.InvoicesAPI.ListInvoices(ctx).Limit(2)
		if cursor != nil {
			r = r.StartingAfter(*cursor)
		}
		l, _, err := r.Execute()
		return l, err
	}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, f.GetId())
	}
	if len(ids) != 3 || ids[2] != "inv_3" {
		t.Errorf("ids = %v", ids)
	}
	ps := b.peticiones()
	if len(ps) != 2 || ps[0].ruta != "/v1/invoices?limit=2" || ps[1].ruta != "/v1/invoices?limit=2&starting_after=inv_2" {
		t.Errorf("rutas = %v", ps)
	}
}
