// Generado por sdks/herramientas/fragmentos.mjs: el fragmento de la documentación, apuntando a Prism. No editar.
package main

import (
	"context"
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

	for item, err := range trevol.Paginate(ctx, func(cursor *string) (*generado.RadianEventList, error) {
		r := c.RadianEventsAPI.ListRadianEvents(ctx).Limit(100)
		if cursor != nil {
			r = r.StartingAfter(*cursor)
		}
		l, _, err := r.Execute()
		return l, err
	}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.GetId())
	}
}
