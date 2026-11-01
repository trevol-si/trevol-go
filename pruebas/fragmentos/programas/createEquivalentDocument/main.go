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

	var entrada generado.EquivalentDocumentInput
	cuerpo := `{
  "external_id": "TIQ-1",
  "customer": {
    "final_consumer": true
  },
  "pos": {
    "register": {
      "plate": "CJ-01",
      "location": "Local 1, Jamundí",
      "cashier": "PEREZ ANA",
      "type": "Caja principal"
    }
  },
  "numbering": {
    "prefix": "POS2"
  },
  "payment": {
    "form": "cash",
    "methods": [
      {
        "code": "10"
      }
    ]
  },
  "items": [
    {
      "description": "Gaseosa 400 ml",
      "code": "GAS-1",
      "unit": "94",
      "quantity": "2",
      "unit_price": "3000",
      "taxes": [
        {
          "code": "01",
          "rate": "19"
        }
      ]
    },
    {
      "description": "Almuerzo",
      "unit": "94",
      "quantity": "1",
      "unit_price": "18000"
    }
  ]
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.EquivalentDocumentsAPI.CreateEquivalentDocument(ctx).
		EquivalentDocumentInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
