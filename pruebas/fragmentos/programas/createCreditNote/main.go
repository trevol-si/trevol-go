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

	var entrada generado.NoteInput
	cuerpo := `{
  "external_id": "DEV-2026-000045",
  "concept": "1",
  "description": "Devolución de una unidad por empaque dañado",
  "invoice": {
    "id": "inv_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
  },
  "items": [
    {
      "description": "Café especial 500 g",
      "code": "CAF-500",
      "unit": "94",
      "quantity": "1",
      "unit_price": "32000",
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

	resultado, _, err := c.CreditNotesAPI.CreateCreditNote(ctx).
		NoteInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
