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

	var entrada generado.NumberingAssociationInput
	cuerpo := `{
  "remove_from_current_software": true
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.NumberingRangesAPI.AssociateNumberingRange(ctx, "nrg_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b").
		NumberingAssociationInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
