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

	var entrada generado.EquivalentDocumentAdjustmentInput
	cuerpo := `{
  "external_id": "TIQ-1-DEV",
  "note_type": "credit",
  "concept": "1",
  "equivalent_document": {
    "id": "pos_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
  },
  "items": [
    {
      "description": "Devolución gaseosa",
      "unit": "94",
      "quantity": "1",
      "unit_price": "3000",
      "taxes": [
        {
          "code": "01",
          "rate": "19"
        }
      ]
    }
  ]
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.EquivalentDocumentAdjustmentsAPI.CreateEquivalentDocumentAdjustment(ctx).
		EquivalentDocumentAdjustmentInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
