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

	var entrada generado.PayrollInput
	cuerpo := `{
  "external_id": "NOM-2026-09-001",
  "period": {
    "hire_date": "2024-03-01",
    "start": "2026-09-01",
    "end": "2026-09-30"
  },
  "pay_period": "5",
  "notes": [
    "Nómina de septiembre de 2026"
  ],
  "employee": {
    "type": "01",
    "subtype": "00",
    "high_risk_pension": false,
    "document_type": "13",
    "document_number": "1000000101",
    "first_surname": "PRUEBA",
    "second_surname": "NOMINA01",
    "first_name": "TRABAJADOR",
    "other_names": "DE PRUEBA",
    "workplace": {
      "country": "CO",
      "municipality_code": "76001",
      "address": "Calle 1 # 1-1"
    },
    "integral_salary": false,
    "contract_type": "2",
    "salary": "1423500",
    "code": "TRAB-01"
  },
  "payment": {
    "form": "1",
    "method": "42",
    "bank": "Banco de Pruebas",
    "account_type": "Ahorros",
    "account_number": "0001234501"
  },
  "payment_dates": [
    "2026-09-30"
  ],
  "earnings": {
    "basic": {
      "days": 30,
      "amount": "1423500"
    },
    "transport": [
      {
        "allowance": "200000"
      }
    ]
  },
  "deductions": {
    "health": {
      "percent": "4",
      "amount": "56940"
    },
    "pension": {
      "percent": "4",
      "amount": "56940"
    }
  }
}`
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil {
		log.Fatal(err)
	}

	resultado, _, err := c.PayrollsAPI.CreatePayroll(ctx).
		PayrollInput(entrada).
		// IdempotencyKey("VENTA-2026-000123"). // opcional: si falta, el SDK pone una y la repite en los reintentos
		Execute()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resultado)
}
