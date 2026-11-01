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

	var entrada generado.SupportDocumentAdjustmentInput
	cuerpo := `{
  "external_id": "OC-991-ANULA",
  "concept": "2",
  "description": "Anulación: la compra de leche se registró dos veces en el ERP",
  "support_document": {
    "id": "sd_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
  },
  "items": [
    {
      "description": "Leche cruda",
      "unit": "LTR",
      "quantity": "120",
      "unit_price": "1800"
    }
  ]
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.SupportDocumentAdjustmentsAPI.PreviewSupportDocumentAdjustment(ctx).
		SupportDocumentAdjustmentInput(entrada).
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
