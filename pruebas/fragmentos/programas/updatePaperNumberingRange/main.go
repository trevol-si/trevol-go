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

	var entrada generado.PaperNumberingRangeUpdate
	cuerpo := `{
  "valid_to": "2028-06-30"
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.PaperNumberingRangesAPI.UpdatePaperNumberingRange(ctx, "pnr_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b").
		PaperNumberingRangeUpdate(entrada).
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
