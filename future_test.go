package trevol

import (
	"context"
	"errors"
	"testing"
)

type casoFuturo struct {
	Nombre    string            `json:"nombre"`
	Operacion string            `json:"operacion"`
	Estado    int               `json:"estado"`
	Cuerpo    any               `json:"cuerpo"`
	Cabeceras map[string]string `json:"cabeceras"`
	Evento    map[string]any    `json:"evento"`
	Esperado  struct {
		ID         string `json:"id"`
		Status     any    `json:"status"`
		Cantidad   *int   `json:"cantidad"`
		HasMore    bool   `json:"has_more"`
		Clase      string `json:"clase"`
		Code       string `json:"code"`
		StatusHTTP int    `json:"-"`
		RequestID  string `json:"requestId"`
		Type       string `json:"type"`
		TieneModel bool   `json:"tieneModelo"`
	} `json:"esperado"`
}

func TestRespuestasDelFuturoNoRompen(t *testing.T) {
	var d struct{ Casos []casoFuturo }
	datos(t, "respuestas-futuras.json", &d)
	vistos := 0
	for _, c := range d.Casos {
		if c.Operacion == "" {
			continue
		}
		vistos++
		t.Run(c.Nombre, func(t *testing.T) {
			tipo := "application/json"
			if c.Estado >= 400 {
				tipo = "application/problem+json"
			}
			b := nuevoBancoConLlave(t, llaveTest, []any{resp{estado: c.Estado, cuerpo: c.Cuerpo, tipo: tipo, cabeceras: c.Cabeceras}}, []Option{WithMaxRetries(0)})
			ctx := context.Background()
			if c.Estado < 400 {
				if c.Operacion == "listInvoices" {
					l, _, err := b.cliente.InvoicesAPI.ListInvoices(ctx).Execute()
					if err != nil {
						t.Fatal(err)
					}
					if len(l.GetData()) != *c.Esperado.Cantidad || l.GetHasMore() != c.Esperado.HasMore {
						t.Errorf("lista = %d %v", len(l.GetData()), l.GetHasMore())
					}
					return
				}
				doc, _, err := b.cliente.InvoicesAPI.RetrieveInvoice(ctx, "inv_x").Execute()
				if err != nil {
					t.Fatal(err)
				}
				if doc.GetId() != c.Esperado.ID || doc.GetStatus() != c.Esperado.Status.(string) {
					t.Errorf("doc = %s %s", doc.GetId(), doc.GetStatus())
				}
				return
			}
			_, _, err := b.cliente.InvoicesAPI.RetrieveInvoice(ctx, "inv_x").Execute()
			var te *Error
			if !errors.As(err, &te) {
				t.Fatalf("err = %v", err)
			}
			if string(te.Kind) != c.Esperado.Clase || te.Code != c.Esperado.Code || te.Status != c.Estado {
				t.Errorf("%s %s %d", te.Kind, te.Code, te.Status)
			}
			if c.Esperado.RequestID != "" && te.RequestID != c.Esperado.RequestID {
				t.Errorf("request id = %s", te.RequestID)
			}
		})
	}
	if vistos == 0 {
		t.Fatal("no hay casos")
	}
}

func TestEventosDelFuturo(t *testing.T) {
	var d struct{ Casos []casoFuturo }
	datos(t, "respuestas-futuras.json", &d)
	for _, c := range d.Casos {
		if c.Evento == nil {
			continue
		}
		t.Run(c.Nombre, func(t *testing.T) {
			secreto, cuerpo, h, ts := firmar(t, c.Evento)
			w := verificadorEn(t, secreto, ts)
			e, err := w.Verify(cuerpo, h)
			if err != nil {
				t.Fatal(err)
			}
			if e.GetType() != c.Esperado.Type {
				t.Errorf("type = %s", e.GetType())
			}
			_, hay, _ := EventData(e)
			if hay != c.Esperado.TieneModel {
				t.Errorf("tieneModelo = %v", hay)
			}
			if len(e.Data) != len(c.Evento["data"].(map[string]any)) {
				t.Error("event.Data no es el mapa completo")
			}
		})
	}
}
