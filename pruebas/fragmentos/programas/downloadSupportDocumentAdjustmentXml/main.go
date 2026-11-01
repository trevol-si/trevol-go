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

	archivo, _, err := c.SupportDocumentAdjustmentsAPI.DownloadSupportDocumentAdjustmentXml(ctx, "sda_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b").Execute()
	if err != nil {
		log.Fatal(err)
	}
	defer archivo.Close()
	fmt.Println(archivo.Name()) // un *os.File temporal con el XML
}
