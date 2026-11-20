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

	var entrada generado.CompanyLogoInput
	cuerpo := `{
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAAAGAAAAAgAQMAAADXDu8XAAAABlBMVEX///8KZKiBydPRAAAAJ0lEQVR42mNgIAPw////AcZmQpagAcf+/////////0BDe+jlH3IAAC07DVBS46jVAAAAAElFTkSuQmCC"
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.CompanySettingsAPI.UpdateCompanyLogo(ctx).
		CompanyLogoInput(entrada).
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
