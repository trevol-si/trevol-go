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

	var entrada generado.NoteInput
	cuerpo := `{
  "external_id": "INT-2026-000012",
  "concept": "1",
  "description": "Intereses de mora de la factura de septiembre",
  "invoice": {
    "id": "inv_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
  },
  "items": [
    {
      "description": "Intereses de mora",
      "unit": "94",
      "quantity": "1",
      "unit_price": "45000"
    }
  ]
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.DebitNotesAPI.PreviewDebitNote(ctx).
		NoteInput(entrada).
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
