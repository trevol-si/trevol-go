# trevol-go

SDK de Go para la **API de Trevol**: facturación electrónica, documento soporte, documento equivalente, nómina electrónica y eventos
RADIAN ante la DIAN (Colombia). Documentación: <https://platform.trevol.si/docs>.

> **In English.** Go SDK for the Trevol API (Colombian e-invoicing, DIAN). Go ≥ 1.23, one dependency
> (`standard-webhooks/libraries`). Typed models and API clients generated from the OpenAPI spec (OpenAPI Generator, package `generado`), plus a
> thin hand-written layer (package `trevol`): automatic `Idempotency-Key` (the same one on retries), retries with exponential backoff honouring
> `Retry-After`, typed errors by problem+json `code` (`*trevol.Error`, use `errors.As`), environment inferred from the key prefix (`tvl_test_` =
> DIAN test environment, `tvl_live_` = production), pinned `Trevol-Version`, Standard Webhooks signature verification, cursor pagination with
> `iter.Seq2` and PDF/XML/ZIP downloads. Docs are in Spanish like the API.

**Versión 0.1.0** — API `2026-10-22`. Licencia MIT.

## Instalación

```sh
go get github.com/trevol/trevol-go
```

Necesita Go 1.23 o más nuevo (usa `iter.Seq2`). La única dependencia es la biblioteca abierta de
[Standard Webhooks](https://github.com/standard-webhooks/standard-webhooks/tree/main/libraries/go) (MIT).

## Uso

La llave de la API se crea en la consola de desarrolladores (<https://platform.trevol.si/llaves>) y se guarda en el entorno de tu **servidor**:
`TREVOL_API_KEY`. Una llave `tvl_test_…` llama a la habilitación de la DIAN; una `tvl_live_…`, a producción. El SDK deduce el entorno del
prefijo (`c.Environment`).

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	trevol "github.com/trevol/trevol-go"
	"github.com/trevol/trevol-go/generado"
)

func main() {
	ctx := context.Background()
	c, err := trevol.NewClient(os.Getenv("TREVOL_API_KEY")) // tvl_test_… → c.Environment == trevol.EnvTest
	if err != nil {
		log.Fatal(err)
	}

	var entrada generado.InvoiceInput
	_ = json.Unmarshal([]byte(`{"external_id": "VENTA-2026-000123", "customer": {"final_consumer": true},
		"items": [{"description": "Café especial 500 g", "quantity": "1", "unit_price": "32000", "taxes": [{"code": "01", "rate": "19"}]}],
		"payment": {"form": "cash", "methods": [{"code": "10"}]}}`), &entrada)

	factura, _, err := c.InvoicesAPI.CreateInvoice(ctx).
		InvoiceInput(entrada).
		IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	var te *trevol.Error
	if errors.As(err, &te) {
		if te.Kind == trevol.KindValidation {
			for _, x := range te.Errors {
				log.Println(x.Pointer, x.Message)
			}
		}
		log.Fatal(te.Code, " ", te.RequestID)
	}
	if factura.GetStatus() == "processing" {
		// 202: la DIAN no contestó; Trevol lo termina y avisa por webhook. NO crees otro documento para la misma venta.
	}

	// Todas las páginas de un listado, perezosamente
	for f, err := range trevol.Paginate(ctx, func(cursor *string) (*generado.InvoiceList, error) {
		r := c.InvoicesAPI.ListInvoices(ctx).Limit(100)
		if cursor != nil {
			r = r.StartingAfter(*cursor)
		}
		l, _, err := r.Execute()
		return l, err
	}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(f.GetId(), f.GetStatus())
	}

	// Los archivos llegan como *os.File temporal (acuérdate de cerrarlo y borrarlo)
	pdf, _, err := c.InvoicesAPI.DownloadInvoicePdf(ctx, "inv_…").Execute()
	_ = pdf
}
```

Los nombres de los métodos son los del `operationId` de la [especificación](https://platform.trevol.si/docs/api) (`CreateInvoice`,
`ListInvoices`, `RetrieveInvoice`, `CorrectInvoice`, `SendInvoice`, `DownloadInvoicePdf`…), con el estilo de petición por pasos del generador
(`.InvoiceInput(entrada).Execute()`). Los campos del cuerpo llevan los **mismos nombres que en la API** en el JSON (`external_id`), y los
importes y las fechas viajan como texto (`"32000.50"`, `"2026-10-31"`). El cuerpo más cómodo es decodificar JSON con `json.Unmarshal`
en el modelo (`generado.InvoiceInput`), como en el ejemplo.

## Lo que conviene saber

### Idempotencia: manda `external_id` y tu propia clave

El SDK genera una `Idempotency-Key` por llamada y **la repite en todos los reintentos de esa llamada**. Pero si tu proceso muere y vuelves a
llamar, saldrá otra clave y, si no mandaste `external_id` (es opcional), podría duplicarse el documento. Por eso:

- manda siempre `external_id` (tu id de la venta), y
- pasa una clave derivada de tu venta: `.IdempotencyKey("VENTA-2026-000123")`.

Ante un `KindConflict` con `Code == "external_id_in_use"`, el documento ya existe: usa `te.Problem["existing_id"]`. Todo `*trevol.Error` lleva la
`IdempotencyKey` que se usó para reintentar más tarde con la misma. Un `KindConnection` también.

### Respuestas 202 (`processing`, `sending`, `validating`, `contingency`)

No son errores ni se reintentan: se devuelven como el modelo de la respuesta. **No crees otro documento**: espera el webhook o consulta con
`RetrieveInvoice`; repetir la misma llamada con la misma `Idempotency-Key` continúa el mismo documento. El PDF y el ZIP llegan después del `201`
(evento `invoice.delivered`, o `delivery.status == "completed"`).

### Reintentos

Hasta 2 reintentos (3 intentos) con espera creciente (0,5 s, 1 s, 2 s… con tope de 8 s y un poco de azar) ante un error de red, 429, 500,
502, 503, 504, un 409 `request_in_progress` o `previous_event_pending`, y todo problema que diga `retryable: true` (salvo el 402). Respeta
`Retry-After`; si pide más de `WithMaxRetryAfter` (60 s) no espera y devuelve el error con `RetryAfter`. Las operaciones sin `Idempotency-Key`
(crear un endpoint, `…/send`, `DELETE`) solo se reintentan con 429. Los 202 y los errores finales (400, 401, 402, 403, 404, 422 y los 409
`invalid_state`, `external_id_in_use`, `duplicate_document`) **nunca** se reintentan. Cancelar el `context` corta la espera y el intento.

### La versión de la API

El SDK fija `Trevol-Version: 2026-10-22` en cada petición. **Ojo:** una llave creada con la versión `2026-10-21` que empiece a usar este SDK
pasa a `2026-10-22` (el encabezado manda): recibe `202 processing` donde antes recibía `503`. Si todavía no quieres ese cambio:
`trevol.NewClient(llave, trevol.WithAPIVersion("2026-10-21"))`. Más en <https://platform.trevol.si/docs/versiones>.

### Errores tipados

Todo lo que no es 2xx llega como `*trevol.Error` (se saca con `errors.As`, también a través del `*url.Error` con que `net/http` lo envuelve) con una
`Kind` que sale del `code` del problem+json: `KindAuthentication`, `KindPermissionDenied`, `KindNotFound`, `KindInvalidRequest`, `KindIdempotency`,
`KindValidation` (con `Errors`: `Field`, `Pointer`, `Message`), `KindConflict`, `KindPrecondition`, `KindDianRejected` (con `Problem["document"]` y
`Problem["rules"]`), `KindBalance`, `KindRateLimit`, `KindDianUnavailable` y `KindInternalServer`. Un `code` que el SDK todavía no conoce llega
como `KindUnknown` con ese `Code`: nunca rompe. Además: `KindConnection` (no hubo respuesta), `KindConfiguration` (mal uso, antes de llamar) y
`KindWebhookVerification`. También sirve `errors.Is(err, &trevol.Error{Kind: trevol.KindNotFound})`.
Cada error tiene `Status`, `Code`, `Type`, `Title`, `Detail`, `RequestID`, `Retryable`, `RetryAfter`, `Errors`, `IdempotencyKey`, `Problem` (el
cuerpo completo) y `Header` (los de la respuesta). Ninguno incluye tu llave. Cuando hay un error, el `*http.Response` que devuelve el método
generado es `nil`: las cabeceras de la respuesta están en `te.Header`.

### Webhooks

```go
w, err := trevol.NewWebhook(os.Getenv("TREVOL_WEBHOOK_SECRET")) // whsec_…
http.HandleFunc("/webhooks/trevol", func(rw http.ResponseWriter, r *http.Request) {
	cuerpo, _ := io.ReadAll(r.Body) // el cuerpo CRUDO
	evento, err := w.Verify(cuerpo, r.Header)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	// Deduplica por evento.GetId() (también llega en la cabecera webhook-id): la verificación no evita repeticiones.
	datos, ok, _ := trevol.EventData(evento) // el modelo según el type (p. ej. *generado.DocumentEventData)
	rw.WriteHeader(http.StatusOK)
})
```

La firma se calcula sobre los **bytes exactos**: lee el cuerpo antes de decodificarlo. La biblioteca de Standard Webhooks verifica la firma (en
tiempo constante, con varias firmas durante la rotación del secreto); el SDK exige que `webhook-timestamp` sea un número de segundos y tolera 5
minutos hacia atrás y hacia adelante.

### Cabeceras propias por llamada

`c.GetConfig().AddDefaultHeader("X-Mi-Cabecera", "…")` agrega una cabecera a todas las peticiones (el SDK siempre pone las suyas encima).
Para un proxy, TLS o métricas, pasa tu `*http.Client` con `trevol.WithHTTPClient(...)`: su `Transport` hace las peticiones y el SDK lo envuelve.

## Opciones

```go
trevol.NewClient(llave,
	trevol.WithAPIVersion("2026-10-22"),    // Trevol-Version en cada petición
	trevol.WithMaxRetries(2),               // 0 los apaga
	trevol.WithTimeout(65*time.Second),     // por intento
	trevol.WithMaxRetryAfter(60*time.Second),
	trevol.WithBaseURL("https://api.trevol.si/v1"),
	trevol.WithEnvironment(trevol.EnvTest), // si la llave no coincide: *Error KindConfiguration antes de llamar
	trevol.WithHTTPClient(miClienteHTTP),
)
```

## Cómo está hecho

- `generado/` lo escribe [OpenAPI Generator](https://openapi-generator.tech) 7.26.0 (`go`) desde la especificación pública y **no se edita**
  (ver `GENERADO.md`).
- Lo propio (`client.go`, `transport.go`, `error.go`, `paginate.go`, `webhook.go`) vive fuera y solo usa el punto de extensión del código
  generado: un `http.RoundTripper` en `Configuration.HTTPClient`, que devuelve el `*trevol.Error`.
- Las pruebas: `go test ./...` (unitarias sin red contra un servidor HTTP local, con los mismos casos de los demás SDK en `pruebas/datos/`);
  `TREVOL_PRISM_URL=http://127.0.0.1:4010 go test ./pruebas/prism ./pruebas/fragmentos` contra un servidor simulado
  (`npx @stoplight/prism-cli mock pruebas/datos/openapi.json -p 4010`) que valida cada petición contra la especificación, y que además compila y
  corre cada fragmento de código de la documentación.
- `go run ./pruebas/humo` es la prueba de humo **real** (solo con una llave `tvl_test_`; no emite documentos).

## Licencia

MIT © 2026 Trevol.
