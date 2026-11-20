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

	var entrada generado.PdfSettingsUpdate
	cuerpo := `{
  "letter_design_id": "pdd_7d3e9a107d3e9a107d3e9a107d3e9a10",
  "receipt_design_id": "pdd_41b2c6f041b2c6f041b2c6f041b2c6f0"
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.CompanySettingsAPI.UpdatePdfSettings(ctx).
		PdfSettingsUpdate(entrada).
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
