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

	var entrada generado.SupportDocumentInput
	cuerpo := `{
  "external_id": "OC-991",
  "seller": {
    "resident": true,
    "person_type": "natural",
    "document_type": "31",
    "document_number": "1020304050",
    "name": "PEREZ GOMEZ ANA MARIA",
    "address": "Vereda El Silencio, finca La Esperanza",
    "municipality_code": "25053",
    "postal_code": "252007",
    "tax_responsibilities": [
      "R-99-PN"
    ]
  },
  "items": [
    {
      "description": "Leche cruda",
      "unit": "LTR",
      "quantity": "120",
      "unit_price": "1800"
    }
  ],
  "payment": {
    "form": "cash",
    "methods": [
      {
        "code": "10"
      }
    ]
  },
  "notes": [
    "Compra de la semana 42 de octubre"
  ]
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.SupportDocumentsAPI.CreateSupportDocument(ctx).
		SupportDocumentInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
