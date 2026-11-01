// Generado por sdks/herramientas/fragmentos.mjs: el fragmento de la documentación, apuntando a Prism. No editar.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	trevol "github.com/trevol/trevol-go"
	"github.com/trevol/trevol-go/generado"
)

func main() {
	ctx := context.Background()
	c, err := trevol.NewClient(os.Getenv("TREVOL_API_KEY"), trevol.WithBaseURL(os.Getenv("TREVOL_PRISM_URL")), trevol.WithMaxRetries(0))
	if err != nil {
		log.Fatal(err)
	}

	var entrada generado.InvoiceInput
	cuerpo := `{
  "external_id": "EXP-77",
  "document_type": "02",
  "currency": {
    "code": "USD",
    "exchange_rate": "3950.12",
    "rate_date": "2026-10-21"
  },
  "delivery_terms": {
    "incoterm": "FOB",
    "conditions": "Puerto de Cartagena"
  },
  "customer": {
    "document_type": "50",
    "document_number": "US123456789",
    "name": "ACME INC",
    "country": "US",
    "city": "Miami",
    "region": "Florida",
    "postal_code": "33131",
    "address": "200 S Biscayne Blvd"
  },
  "items": [
    {
      "description": "Consultoría",
      "quantity": "1",
      "unit_price": "1000",
      "taxes": [
        {
          "code": "01",
          "excluded": true
        }
      ]
    }
  ],
  "payment": {
    "form": "credit",
    "term_days": 30
  }
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.InvoicesAPI.CreateInvoice(ctx).
		InvoiceInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
