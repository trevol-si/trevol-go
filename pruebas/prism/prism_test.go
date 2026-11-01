// Contra Prism (servidor simulado desde docs/api/openapi.json, docs/api/43 §2.10.2):
//
//	sdks/herramientas/prism.sh -- bash -c 'cd sdks/go && go test ./pruebas/prism/'
//
// Prism valida cada petición contra la especificación PÚBLICA: si el SDK serializa mal un cuerpo, olvida una cabecera o la
// Idempotency-Key, responde 4xx y la prueba falla (el motivo viene en la cabecera `sl-violations`, que el *trevol.Error conserva).
package prism

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	trevol "github.com/trevol-si/trevol-go"
	"github.com/trevol-si/trevol-go/generado"
)

var ctx = context.Background()

func cliente(t *testing.T, opts ...trevol.Option) *trevol.Client {
	t.Helper()
	url := os.Getenv("TREVOL_PRISM_URL")
	if url == "" {
		t.Skip("necesita Prism: sdks/herramientas/prism.sh -- bash -c 'cd sdks/go && go test ./pruebas/prism/'")
	}
	llave := os.Getenv("TREVOL_API_KEY")
	if llave == "" {
		llave = "tvl_test_" + strings.Repeat("a1", 32)
	}
	c, err := trevol.NewClient(llave, append([]trevol.Option{trevol.WithBaseURL(url), trevol.WithMaxRetries(0)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// ejemplo lee el cuerpo de ejemplo de la especificación (pruebas/datos/operaciones.json) y lo decodifica en el modelo generado.
func ejemplo(t *testing.T, op, nombre string, destino any) {
	t.Helper()
	crudo, err := os.ReadFile(filepath.Join("..", "datos", "operaciones.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ops []struct {
		OperationID string `json:"operationId"`
		Ejemplos    []struct {
			Nombre string          `json:"nombre"`
			Valor  json.RawMessage `json:"valor"`
		} `json:"ejemplos"`
	}
	if err := json.Unmarshal(crudo, &ops); err != nil {
		t.Fatal(err)
	}
	for _, o := range ops {
		if o.OperationID != op {
			continue
		}
		for _, e := range o.Ejemplos {
			if nombre == "" || e.Nombre == nombre {
				if err := json.Unmarshal(e.Valor, destino); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
	}
	t.Fatalf("no hay ejemplo %q de %s", nombre, op)
}

// violaciones agrega al fallo el motivo por el que Prism rechazó la petición.
func violaciones(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var te *trevol.Error
	if errors.As(err, &te) {
		if v := te.Header.Get("Sl-Violations"); v != "" {
			t.Fatalf("Prism rechazó la petición (%d): %s", te.Status, v)
		}
	}
	t.Fatal(err)
}

func TestCreaUnaFactura201ConElCuerpoDeEjemplo(t *testing.T) {
	c := cliente(t)
	var entrada generado.InvoiceInput
	ejemplo(t, "createInvoice", "", &entrada)
	doc, _, err := c.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).Execute()
	violaciones(t, err)
	if doc.GetObject() != "invoice" {
		t.Errorf("object = %q", doc.GetObject())
	}
}

func TestCreaUnaNotaCreditoYCalculaUnPreview(t *testing.T) {
	c := cliente(t)
	var nota generado.NoteInput
	ejemplo(t, "createCreditNote", "", &nota)
	nc, _, err := c.CreditNotesAPI.CreateCreditNote(ctx).NoteInput(nota).Execute()
	violaciones(t, err)
	var entrada generado.InvoiceInput
	ejemplo(t, "previewInvoice", "", &entrada)
	p, _, err := c.InvoicesAPI.PreviewInvoice(ctx).InvoiceInput(entrada).Execute()
	violaciones(t, err)
	if nc == nil || p.GetObject() != "preview" || p.Totals == nil {
		t.Errorf("nc=%v preview=%v", nc, p)
	}
}

func TestCorrigeUnaFacturaRechazadaConPut(t *testing.T) {
	c := cliente(t)
	var entrada generado.InvoiceInput
	ejemplo(t, "correctInvoice", "", &entrada)
	d, _, err := c.InvoicesAPI.CorrectInvoice(ctx, "inv_"+strings.Repeat("0", 32)).InvoiceInput(entrada).Execute()
	violaciones(t, err)
	if d == nil {
		t.Error("sin documento")
	}
}

func TestListaFacturasYEventosEItera(t *testing.T) {
	c := cliente(t)
	n := 0
	for f, err := range trevol.Paginate(ctx, func(cursor *string) (*generado.InvoiceList, error) {
		r := c.InvoicesAPI.ListInvoices(ctx).Limit(2)
		if cursor != nil {
			r = r.StartingAfter(*cursor)
		}
		l, _, err := r.Execute()
		return l, err
	}) {
		violaciones(t, err)
		if f.GetId() == "" {
			t.Error("sin id")
		}
		if n++; n == 3 { // Prism repite siempre la misma página con has_more=true (C8): el tope es de la prueba
			break
		}
	}
	if n == 0 {
		t.Error("no hubo facturas")
	}
	n = 0
	for e, err := range trevol.Paginate(ctx, func(cursor *string) (*generado.EventList, error) {
		r := c.EventsAPI.ListEvents(ctx)
		if cursor != nil {
			r = r.StartingAfter(*cursor)
		}
		l, _, err := r.Execute()
		return l, err
	}) {
		violaciones(t, err)
		_ = e
		if n++; n == 3 {
			break
		}
	}
	if n == 0 {
		t.Error("no hubo eventos")
	}
}

func TestDescargaPDFXMLZIPYRespuestaDianLosBytesLleganYElTipoEsElCorrecto(t *testing.T) {
	c := cliente(t)
	id := "inv_" + strings.Repeat("0", 32)
	casos := map[string]func() (*os.File, error){
		"pdf": func() (*os.File, error) {
			f, _, err := c.InvoicesAPI.DownloadInvoicePdf(ctx, id).Execute()
			return f, err
		},
		"xml": func() (*os.File, error) {
			f, _, err := c.InvoicesAPI.DownloadInvoiceXml(ctx, id).Execute()
			return f, err
		},
		"zip": func() (*os.File, error) {
			f, _, err := c.InvoicesAPI.DownloadInvoiceZip(ctx, id).Execute()
			return f, err
		},
		"dian": func() (*os.File, error) {
			f, _, err := c.InvoicesAPI.DownloadInvoiceDianResponse(ctx, id).Execute()
			return f, err
		},
	}
	for nombre, f := range casos {
		archivo, err := f()
		violaciones(t, err)
		bytes, _ := io.ReadAll(archivo)
		archivo.Close()
		os.Remove(archivo.Name())
		if len(bytes) == 0 {
			t.Errorf("%s: sin bytes", nombre)
		}
	}
	_, resp, err := c.InvoicesAPI.DownloadInvoicePdf(ctx, id).Execute()
	violaciones(t, err)
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/pdf") {
		t.Errorf("content-type = %q", resp.Header.Get("Content-Type"))
	}
}

func conPrefer(t *testing.T, valor string) *trevol.Client {
	c := cliente(t)
	c.GetConfig().AddDefaultHeader("Prefer", valor)
	return c
}

func TestPreferCode422SobreUnEventoRadianEsPreconditionErrorTacitAcceptanceNotDue(t *testing.T) {
	c := conPrefer(t, "code=422, example=tacita")
	var entrada generado.RadianEventInput
	ejemplo(t, "createRadianEvent", "tacita", &entrada)
	_, _, err := c.RadianEventsAPI.CreateRadianEvent(ctx).RadianEventInput(entrada).Execute()
	var te *trevol.Error
	if !errors.As(err, &te) || te.Kind != trevol.KindPrecondition || te.Code != "tacit_acceptance_not_due" || te.Status != 422 {
		t.Fatalf("err = %v", err)
	}
}

func TestPreferCode409SobreUnEventoRadianEsConflictError(t *testing.T) {
	c := conPrefer(t, "code=409, example=duplicado")
	var entrada generado.RadianEventInput
	ejemplo(t, "createRadianEvent", "acuse", &entrada)
	_, _, err := c.RadianEventsAPI.CreateRadianEvent(ctx).RadianEventInput(entrada).Execute()
	var te *trevol.Error
	if !errors.As(err, &te) || te.Kind != trevol.KindConflict || te.Status != 409 {
		t.Fatalf("err = %v", err)
	}
	if id, _ := te.Problem["existing_id"].(string); !strings.HasPrefix(id, "rev_") {
		t.Errorf("existing_id = %v", te.Problem["existing_id"])
	}
}

func TestPreferCode429EsUnErrorConStatus429RetryableYLaClave(t *testing.T) {
	c := conPrefer(t, "code=429")
	var entrada generado.InvoiceInput
	ejemplo(t, "createInvoice", "", &entrada)
	_, _, err := c.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).Execute()
	var te *trevol.Error
	if !errors.As(err, &te) || te.Status != 429 || !te.Retryable || te.IdempotencyKey == "" {
		t.Fatalf("err = %#v", te)
	}
}

func TestCreaUnEndpointDeWebhooksYMandaLaPrueba202(t *testing.T) {
	c := cliente(t)
	var entrada generado.WebhookEndpointInput
	ejemplo(t, "createWebhookEndpoint", "", &entrada)
	ep, _, err := c.WebhookEndpointsAPI.CreateWebhookEndpoint(ctx).WebhookEndpointInput(entrada).Execute()
	violaciones(t, err)
	entrega, _, err := c.WebhookEndpointsAPI.TestWebhookEndpoint(ctx, "we_"+strings.Repeat("0", 32)).Execute()
	violaciones(t, err)
	if ep.GetObject() != "webhook_endpoint" || entrega == nil {
		t.Errorf("ep=%v entrega=%v", ep, entrega)
	}
}

func TestCompanyYAcquirerDevuelvenModelosNoVoid(t *testing.T) {
	c := cliente(t)
	co, _, err := c.GeneralAPI.RetrieveCompany(ctx).Execute()
	violaciones(t, err)
	a, _, err := c.GeneralAPI.RetrieveDianAcquirer(ctx).DocumentType("31").DocumentNumber("900373591").Execute()
	violaciones(t, err)
	if co.GetObject() != "company" || a.GetObject() != "acquirer" {
		t.Errorf("company=%q acquirer=%q", co.GetObject(), a.GetObject())
	}
}

func TestReintentaContraPrismConLaMismaClaveYAgotaConElErrorDel503(t *testing.T) {
	c := conPrefer(t, "code=503")
	var entrada generado.InvoiceInput
	ejemplo(t, "createInvoice", "", &entrada)
	// WithMaxRetries(1) y esperas cortas: un reintento con la misma clave, y se agota.
	c2, err := trevol.NewClient("", trevol.WithBaseURL(os.Getenv("TREVOL_PRISM_URL")), trevol.WithMaxRetries(1))
	if err != nil {
		t.Fatal(err)
	}
	c2.GetConfig().AddDefaultHeader("Prefer", "code=503")
	_ = c
	_, _, err = c2.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).IdempotencyKey("PRUEBA-PRISM-1").Execute()
	var te *trevol.Error
	if !errors.As(err, &te) || te.Status != 503 || te.IdempotencyKey != "PRUEBA-PRISM-1" {
		t.Fatalf("err = %v", err)
	}
}
