// Generado por sdks/herramientas/fragmentos.mjs: el fragmento de la documentación, apuntando a Prism. No editar.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	trevol "github.com/trevol-si/trevol-go"
)

func main() {
	ctx := context.Background()
	c, err := trevol.NewClient(os.Getenv("TREVOL_API_KEY"), trevol.WithBaseURL(os.Getenv("TREVOL_PRISM_URL")), trevol.WithMaxRetries(0))
	if err != nil {
		log.Fatal(err)
	}

	archivo, _, err := c.ReportsAPI.ExportDocuments(ctx).
		Format("csv").
		IssuedFrom("2026-09-01").
		IssuedTo("2026-10-31").
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	defer archivo.Close()
	fmt.Println(archivo.Name()) // un *os.File temporal con el CSV
}
