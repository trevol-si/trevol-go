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

	var entrada generado.CatalogCustomerInput
	cuerpo := `{
  "external_id": "ERP-CLI-0042",
  "document_type": "31",
  "document_number": "900373427",
  "name": "DISTRIBUIDORA LOS ANDES SAS",
  "person_type": "legal",
  "tax_responsibilities": [
    "O-13"
  ],
  "email": "facturacion@distribuidoralosandes.example",
  "phone": "6045550142",
  "address": "Carrera 43A # 1-50, Oficina 302",
  "city": "Medellín",
  "municipality_code": "05001",
  "defaults": {
    "payment_form": "credit",
    "term_days": 30,
    "payment_method": "47",
    "withholding_agent": true
  }
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.CustomersAPI.CreateCustomer(ctx).
		CatalogCustomerInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
