// Generado por sdks/herramientas/fragmentos.mjs: el fragmento de la documentación, apuntando a Prism. No editar.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	trevol "github.com/trevol/trevol-go"
)

func main() {
	ctx := context.Background()
	c, err := trevol.NewClient(os.Getenv("TREVOL_API_KEY"), trevol.WithBaseURL(os.Getenv("TREVOL_PRISM_URL")), trevol.WithMaxRetries(0))
	if err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.InvoicesAPI.RetrieveInvoiceTacitAcceptance(ctx, "inv_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b").Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
