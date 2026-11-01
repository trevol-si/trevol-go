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

	var entrada generado.EventResendInput
	cuerpo := `{
  "endpoint_id": "we_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.EventsAPI.ResendEvent(ctx, "evt_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b").
		EventResendInput(entrada).
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
