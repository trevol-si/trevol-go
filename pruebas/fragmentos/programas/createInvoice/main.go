// Generado por sdks/herramientas/fragmentos.mjs: el fragmento de la documentación, apuntando a Prism. No editar.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	trevol "github.com/trevol-si/trevol-go"
	"github.com/trevol-si/trevol-go/generado"
)

func main() {
	ctx := context.Background()
	c, err := trevol.NewClient(os.Getenv("TREVOL_API_KEY"), trevol.WithBaseURL(os.Getenv("TREVOL_PRISM_URL")), trevol.WithMaxRetries(0))
	if err != nil {
		log.Fatal(err)
	}

	var entrada generado.InvoiceInput
	cuerpo := `{
  "external_id": "VENTA-2026-000123",
  "customer": {
    "document_type": "31",
    "document_number": "900123456",
    "name": "COMERCIALIZADORA ANDINA SAS",
    "person_type": "legal",
    "email": "facturacion@comercializadoraandina.example",
    "address": "Calle 72 # 10-34, Oficina 501",
    "municipality_code": "11001"
  },
  "items": [
    {
      "code": "LIC-ANUAL",
      "description": "Licencia anual de software",
      "quantity": "2",
      "unit_price": "150000",
      "taxes": [
        {
          "code": "01",
          "rate": "19"
        }
      ]
    },
    {
      "code": "SOP-MES",
      "description": "Soporte técnico mensual",
      "quantity": "1",
      "unit_price": "80000",
      "taxes": [
        {
          "code": "01",
          "rate": "19"
        }
      ]
    }
  ],
  "payment": {
    "form": "credit",
    "term_days": 30
  },
  "notes": [
    "Gracias por su compra"
  ]
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
