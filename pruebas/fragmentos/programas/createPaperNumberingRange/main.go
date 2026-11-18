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

	var entrada generado.PaperNumberingRangeInput
	cuerpo := `{
  "resolution_number": "18764000004521",
  "prefix": "TP",
  "from": 1,
  "to": 5000,
  "valid_from": "2026-01-15",
  "valid_to": "2028-01-15"
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.PaperNumberingRangesAPI.CreatePaperNumberingRange(ctx).
		PaperNumberingRangeInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
