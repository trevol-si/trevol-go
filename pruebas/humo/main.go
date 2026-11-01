// Prueba de humo REAL en habilitación (docs/api/43 §2.10.4). NO es parte de la suite: la corre una persona, con una llave de PRUEBAS.
//
//	TREVOL_API_KEY=tvl_test_… go run ./pruebas/humo
//
// Lecturas, un preview y un error tipado (validation_error); con TREVOL_HUMO_EMITIR=1 además EMITE una factura en habilitación, espera a que
// quede final y descarga el PDF. Se NIEGA a correr si la llave no es tvl_test_. TREVOL_BASE_URL (opcional) apunta a otro servidor, p. ej. un
// Prism local.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	trevol "github.com/trevol-si/trevol-go"
	"github.com/trevol-si/trevol-go/generado"
)

func main() {
	llave := os.Getenv("TREVOL_API_KEY")
	if !regexp.MustCompile(`^tvl_test_[0-9a-f]{64}$`).MatchString(llave) {
		fmt.Fprintln(os.Stderr, "humo: me niego a correr si TREVOL_API_KEY no es una llave de PRUEBAS (tvl_test_…). Nunca se prueba con tvl_live_.")
		os.Exit(2)
	}
	opts := []trevol.Option{trevol.WithEnvironment(trevol.EnvTest)}
	if u := os.Getenv("TREVOL_BASE_URL"); u != "" {
		opts = append(opts, trevol.WithBaseURL(u))
	}
	c, err := trevol.NewClient(llave, opts...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "humo:", err)
		os.Exit(2)
	}
	ctx := context.Background()
	fallas, total := 0, 0
	paso := func(nombre string, f func() (string, error)) {
		total++
		detalle, err := f()
		if err != nil {
			fallas++
			fmt.Printf("FALLA %s — %v\n", nombre, err)
			return
		}
		if detalle != "" {
			detalle = " — " + detalle
		}
		fmt.Printf("ok   %s%s\n", nombre, detalle)
	}

	paso("GetHealth", func() (string, error) {
		_, err := c.GeneralAPI.GetHealth(ctx).Execute()
		return "", err
	})
	paso("RetrieveCompany", func() (string, error) {
		co, _, err := c.GeneralAPI.RetrieveCompany(ctx).Execute()
		if err != nil {
			return "", err
		}
		if co.GetEnvironment() != "test" {
			return "", fmt.Errorf("la empresa dice environment=%s", co.GetEnvironment())
		}
		return fmt.Sprintf("%s (%s)", co.GetName(), co.GetNit()), nil
	})
	factura := func(externo string, conLineas bool) (generado.InvoiceInput, error) {
		var entrada generado.InvoiceInput
		lineas := `[]`
		if conLineas {
			lineas = `[{"description":"Prueba de humo del SDK de Trevol","quantity":"1","unit_price":"1000","taxes":[{"code":"01","rate":"19"}]}]`
		}
		cuerpo := `{"external_id":"` + externo + `","customer":{"final_consumer":true},"items":` + lineas +
			`,"payment":{"form":"cash","methods":[{"code":"10"}]}}`
		err := json.Unmarshal([]byte(cuerpo), &entrada)
		return entrada, err
	}
	paso("PreviewInvoice", func() (string, error) {
		entrada, err := factura("HUMO-PREVIEW", true)
		if err != nil {
			return "", err
		}
		p, _, err := c.InvoicesAPI.PreviewInvoice(ctx).InvoiceInput(entrada).Execute()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("total %v", p.Totals["total"]), nil
	})
	paso("PreviewInvoice sin líneas → KindValidation (code validation_error)", func() (string, error) {
		entrada, err := factura("HUMO-INVALIDA", false)
		if err != nil {
			return "", err
		}
		_, _, err = c.InvoicesAPI.PreviewInvoice(ctx).InvoiceInput(entrada).Execute()
		var te *trevol.Error
		if !errors.As(err, &te) || te.Kind != trevol.KindValidation || te.Code != "validation_error" {
			return "", fmt.Errorf("no lanzó KindValidation con validation_error: %v", err)
		}
		puntero := ""
		if len(te.Errors) > 0 {
			puntero = te.Errors[0].Pointer
		}
		return fmt.Sprintf("%d %s %s", te.Status, te.Code, puntero), nil
	})
	paso("ListInvoices + Paginate (3)", func() (string, error) {
		if _, _, err := c.InvoicesAPI.ListInvoices(ctx).Limit(3).Execute(); err != nil {
			return "", err
		}
		n := 0
		for _, err := range trevol.Paginate(ctx, func(cursor *string) (*generado.InvoiceList, error) {
			r := c.InvoicesAPI.ListInvoices(ctx).Limit(3)
			if cursor != nil {
				r = r.StartingAfter(*cursor)
			}
			l, _, err := r.Execute()
			return l, err
		}) {
			if err != nil {
				return "", err
			}
			if n++; n == 3 {
				break
			}
		}
		return fmt.Sprintf("%d facturas", n), nil
	})
	paso("RetrieveInvoice inexistente → KindNotFound con request_id", func() (string, error) {
		_, _, err := c.InvoicesAPI.RetrieveInvoice(ctx, "inv_"+strings.Repeat("0", 32)).Execute()
		var te *trevol.Error
		if !errors.As(err, &te) || te.Kind != trevol.KindNotFound || !strings.HasPrefix(te.RequestID, "req_") {
			return "", fmt.Errorf("no lanzó KindNotFound con request_id: %v", err)
		}
		return te.RequestID, nil
	})
	paso("Webhook.Verify (firmado ahora, con un secreto local)", func() (string, error) {
		clave := []byte("trevol-humo-secreto-de-32-bytes!")
		id := "evt_" + strings.Repeat("1", 32)
		cuerpo, _ := json.Marshal(map[string]any{"id": id, "object": "event", "type": "webhook.test", "timestamp": time.Now().UTC().Format(time.RFC3339),
			"environment": "test", "api_version": "2026-10-22", "data": map[string]any{}})
		ts := time.Now().Unix()
		m := hmac.New(sha256.New, clave)
		fmt.Fprintf(m, "%s.%d.%s", id, ts, cuerpo)
		h := http.Header{}
		h.Set("webhook-id", id)
		h.Set("webhook-timestamp", fmt.Sprint(ts))
		h.Set("webhook-signature", "v1,"+base64.StdEncoding.EncodeToString(m.Sum(nil)))
		w, err := trevol.NewWebhook("whsec_" + base64.StdEncoding.EncodeToString(clave))
		if err != nil {
			return "", err
		}
		e, err := w.Verify(cuerpo, h)
		if err != nil {
			return "", err
		}
		return e.GetType(), nil
	})

	if os.Getenv("TREVOL_HUMO_EMITIR") == "1" {
		id := ""
		paso("CreateInvoice (EMITE en habilitación)", func() (string, error) {
			entrada, err := factura(fmt.Sprintf("HUMO-GO-%d", time.Now().Unix()), true)
			if err != nil {
				return "", err
			}
			d, _, err := c.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).Execute()
			if err != nil {
				return "", err
			}
			id = d.GetId()
			return id + " " + d.GetStatus(), nil
		})
		if id != "" {
			paso("espera un estado final (hasta 90 s)", func() (string, error) {
				limite := time.Now().Add(90 * time.Second)
				for {
					d, _, err := c.InvoicesAPI.RetrieveInvoice(ctx, id).Execute()
					if err != nil {
						return "", err
					}
					switch estado := d.GetStatus(); estado {
					case "pending", "sending", "processing":
						if time.Now().After(limite) {
							return "", fmt.Errorf("status = %s", estado)
						}
						time.Sleep(3 * time.Second)
					case "accepted":
						return estado, nil
					default:
						return "", fmt.Errorf("status = %s", estado)
					}
				}
			})
			paso("DownloadInvoicePdf", func() (string, error) {
				archivo, _, err := c.InvoicesAPI.DownloadInvoicePdf(ctx, id).Execute()
				if err != nil {
					return "", err
				}
				defer archivo.Close()
				if _, err := archivo.Seek(0, io.SeekStart); err != nil {
					return "", err
				}
				bytes, err := io.ReadAll(archivo)
				if err != nil {
					return "", err
				}
				if !strings.HasPrefix(string(bytes), "%PDF") {
					return "", fmt.Errorf("no es un PDF")
				}
				return fmt.Sprintf("PDF %d bytes", len(bytes)), nil
			})
		}
	} else {
		fmt.Println("(no se emite nada: define TREVOL_HUMO_EMITIR=1 para emitir una factura en habilitación)")
	}

	fmt.Printf("\n%d/%d pasos bien\n", total-fallas, total)
	if fallas > 0 {
		os.Exit(1)
	}
}
