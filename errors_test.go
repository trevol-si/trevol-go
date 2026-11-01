package trevol

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/trevol/trevol-go/generado"
)

var estadoPorClase = map[string]int{"AuthenticationError": 401, "PermissionDeniedError": 403, "NotFoundError": 404, "InvalidRequestError": 400,
	"IdempotencyError": 409, "ValidationError": 422, "ConflictError": 409, "PreconditionError": 422, "DianRejectedError": 422,
	"BalanceError": 402, "RateLimitError": 429, "DianUnavailableError": 503, "InternalServerError": 500}

func leerFactura(b *banco) (*generado.Document, error) {
	d, _, err := b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	return d, err
}

func comoError(t *testing.T, err error) *Error {
	t.Helper()
	var te *Error
	if !errors.As(err, &te) {
		t.Fatalf("no es un *trevol.Error: %T %v", err, err)
	}
	return te
}

func TestCadaCodeVaASuClaseATravesDeUnMetodoGenerado(t *testing.T) {
	var errores struct {
		Clases map[string][]string `json:"clases"`
	}
	datos(t, "errores.json", &errores)
	total := 0
	for clase, codes := range errores.Clases {
		for _, code := range codes {
			total++
			t.Run(code, func(t *testing.T) {
				b := nuevoBancoConLlave(t, llaveTest, []any{problemaResp(estadoPorClase[clase], code, map[string]any{"retryable": false}, nil)}, []Option{WithMaxRetries(0)})
				_, err := leerFactura(b)
				e := comoError(t, err)
				if string(e.Kind) != clase || e.Code != code || e.Status != estadoPorClase[clase] {
					t.Errorf("kind=%s code=%s status=%d", e.Kind, e.Code, e.Status)
				}
				if e.RequestID != "req_c3c3c3c3c3c3c3c3c3c3c3c3" || e.Type != "https://platform.trevol.si/docs/errores#"+code || e.Problem["code"] != code {
					t.Errorf("campos: %+v", e)
				}
				if !errors.Is(err, &Error{Kind: e.Kind}) {
					t.Error("errors.Is por clase")
				}
			})
		}
	}
	if total != 30 {
		t.Errorf("hay %d codes, esperados 30", total)
	}
}

func TestElErrorNoSeEnvuelveEnGenericOpenAPIError(t *testing.T) {
	b := nuevoBanco(t, problemaResp(404, "not_found", nil, nil))
	_, err := leerFactura(b)
	var g *generado.GenericOpenAPIError
	if errors.As(err, &g) {
		t.Error("el error del SDK se envolvió en GenericOpenAPIError")
	}
	if comoError(t, err).Kind != KindNotFound {
		t.Error("clase")
	}
}

func TestUnCodeDesconocidoEsLaClaseBaseYNoRompe(t *testing.T) {
	b := nuevoBanco(t, problemaResp(422, "code_del_futuro", nil, nil))
	e := comoError(t, mustErr(leerFactura(b)))
	if e.Kind != KindUnknown || e.Code != "code_del_futuro" {
		t.Errorf("%+v", e)
	}
}

func mustErr(_ *generado.Document, err error) error { return err }

func TestUnaRespuestaQueNoEsJSONDaLaClasePorEstadoConCodeVacioYElCuerpoRecortado(t *testing.T) {
	html := []byte("<html>" + strings.Repeat("x", 5000) + "</html>")
	for estado, clase := range map[int]Kind{502: KindInternalServer, 404: KindNotFound, 409: KindConflict, 401: KindAuthentication} {
		b := nuevoBancoConLlave(t, llaveTest, []any{resp{estado: estado, crudo: html, tipo: "text/html"}}, []Option{WithMaxRetries(0)})
		e := comoError(t, mustErr(leerFactura(b)))
		if e.Kind != clase || e.Code != "" || len(e.Detail) != 2048 {
			t.Errorf("%d: kind=%s code=%q detail=%d", estado, e.Kind, e.Code, len(e.Detail))
		}
	}
}

func TestTomaElRequestIDDeLaCabeceraCuandoElCuerpoNoLoTrae(t *testing.T) {
	cuerpo := map[string]any{"type": "x", "title": "t", "status": 404, "code": "not_found", "retryable": false}
	b := nuevoBanco(t, resp{estado: 404, cuerpo: cuerpo, tipo: "application/problem+json", cabeceras: map[string]string{"Request-Id": "req_d4d4d4d4d4d4d4d4d4d4d4d4"}})
	if e := comoError(t, mustErr(leerFactura(b))); e.RequestID != "req_d4d4d4d4d4d4d4d4d4d4d4d4" {
		t.Errorf("request id = %q", e.RequestID)
	}
}

func TestValidationErrorTraeErrorsConFieldPointerYMessage(t *testing.T) {
	b := nuevoBanco(t, problemaResp(422, "validation_error", map[string]any{"errors": []any{map[string]any{"field": "items", "pointer": "/items", "message": "Falta"}}}, nil))
	e := comoError(t, mustErr(leerFactura(b)))
	if len(e.Errors) != 1 || e.Errors[0].Field != "items" || e.Errors[0].Pointer != "/items" || e.Errors[0].Message != "Falta" {
		t.Errorf("%+v", e.Errors)
	}
}

func TestLosCamposExtraQuedanEnProblem(t *testing.T) {
	b := nuevoBanco(t, problemaResp(409, "external_id_in_use", map[string]any{"existing_id": "inv_x"}, nil))
	if e := comoError(t, mustErr(leerFactura(b))); e.Problem["existing_id"] != "inv_x" || e.Kind != KindConflict {
		t.Errorf("%+v", e.Problem)
	}
}

func TestLaLlaveNoApareceEnErrorNiEnFormatoDeNingunError(t *testing.T) {
	b := nuevoBanco(t, problemaResp(401, "invalid_api_key", nil, nil))
	err := mustErr(leerFactura(b))
	e := comoError(t, err)
	textos := []string{err.Error(), e.Error(), fmt.Sprintf("%v", e), fmt.Sprintf("%+v", err), fmt.Sprintf("%#v", e.Problem)}
	for _, tx := range textos {
		if strings.Contains(tx, llaveTest) || strings.Contains(tx, strings.Repeat("a1", 32)) {
			t.Errorf("la llave aparece en %q", tx)
		}
	}
	if !strings.Contains(e.Error(), "invalid_api_key") || !strings.Contains(e.Error(), "HTTP 401") || !strings.Contains(e.Error(), "req_") {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestUnFalloDeRedEsAPIConnectionConLaClaveYSeReintentaSoloSiEsSeguro(t *testing.T) {
	// POST /invoices lleva clave: reintenta (3 intentos); POST /webhook_endpoints no: un solo intento.
	b := nuevoBanco(t, resp{cortar: true})
	_, err := crearFactura(t, b)
	e := comoError(t, err)
	if e.Kind != KindConnection || e.IdempotencyKey == "" || len(b.peticiones()) != 3 {
		t.Errorf("kind=%s clave=%q intentos=%d", e.Kind, e.IdempotencyKey, len(b.peticiones()))
	}
	b2 := nuevoBanco(t, resp{cortar: true})
	_, _, err = b2.cliente.WebhookEndpointsAPI.CreateWebhookEndpoint(context.Background()).WebhookEndpointInput(*generado.NewWebhookEndpointInput("https://tu-servidor.co/h", []string{"*"})).Execute()
	if comoError(t, err).Kind != KindConnection || len(b2.peticiones()) != 1 {
		t.Errorf("intentos = %d", len(b2.peticiones()))
	}
}

func TestUn429ConRetryAfterMayorQueElMaximoNoEsperaYLanzaElErrorConRetryAfter(t *testing.T) {
	b := nuevoBanco(t, problemaResp(429, "rate_limited", map[string]any{"retryable": true}, map[string]string{"Retry-After": "120"}))
	e := comoError(t, mustErr(leerFactura(b)))
	if e.Kind != KindRateLimit || e.RetryAfter.Seconds() != 120 || len(b.esperas) != 0 || len(b.peticiones()) != 1 {
		t.Errorf("%+v esperas=%v", e, b.esperas)
	}
}

func TestUn202EsUnResultadoNoUnErrorNiUnReintento(t *testing.T) {
	doc := map[string]any{"id": idDoc, "object": "invoice", "status": "processing"}
	b := nuevoBanco(t, jsonResp(202, doc))
	d, _, err := b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	if err != nil || d.GetStatus() != "processing" || len(b.peticiones()) != 1 {
		t.Errorf("%v %v", d, err)
	}
}
