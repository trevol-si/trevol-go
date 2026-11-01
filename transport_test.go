package trevol

import (
	"math"
	"testing"
	"time"
)

type casoTransporte struct {
	Nombre        string  `json:"nombre"`
	Reintento     string  `json:"reintento"`
	Intento       int     `json:"intento"`
	MaxReintentos int     `json:"maxReintentos"`
	MaxRetryAfter float64 `json:"maxRetryAfter"`
	Azar          float64 `json:"azar"`
	Resultado     struct {
		Tipo       string   `json:"tipo"`
		Estado     int      `json:"estado"`
		Code       string   `json:"code"`
		Retryable  *bool    `json:"retryable"`
		RetryAfter *float64 `json:"retryAfter"`
	} `json:"resultado"`
	Esperado struct {
		Reintentar bool     `json:"reintentar"`
		Espera     *float64 `json:"espera"`
		Motivo     *string  `json:"motivo"`
	} `json:"esperado"`
}

func segundos(f float64) time.Duration { return time.Duration(f * float64(time.Second)) }

func TestHayCasos(t *testing.T) {
	var d struct{ Casos []casoTransporte }
	datos(t, "transporte.json", &d)
	if len(d.Casos) < 50 {
		t.Fatalf("hay %d casos", len(d.Casos))
	}
}

func TestDecidirConPruebasDatosTransporteJSON(t *testing.T) {
	var d struct{ Casos []casoTransporte }
	datos(t, "transporte.json", &d)
	for _, c := range d.Casos {
		t.Run(c.Nombre, func(t *testing.T) {
			o := Outcome{Network: c.Resultado.Tipo == "red", Status: c.Resultado.Estado, Code: c.Resultado.Code, Retryable: c.Resultado.Retryable}
			if c.Resultado.RetryAfter != nil {
				o.RetryAfter = segundos(*c.Resultado.RetryAfter)
			}
			got := Decide(c.Intento, c.Reintento, c.MaxReintentos, segundos(c.MaxRetryAfter), c.Azar, o)
			if got.Retry != c.Esperado.Reintentar {
				t.Fatalf("Retry = %v, esperado %v", got.Retry, c.Esperado.Reintentar)
			}
			if c.Esperado.Espera != nil && math.Abs(got.Wait.Seconds()-*c.Esperado.Espera) > 1e-6 {
				t.Fatalf("Wait = %v, esperado %vs", got.Wait, *c.Esperado.Espera)
			}
			if c.Esperado.Motivo != nil && got.Reason != *c.Esperado.Motivo {
				t.Fatalf("Reason = %q, esperado %q", got.Reason, *c.Esperado.Motivo)
			}
			if !c.Esperado.Reintentar && got.Wait != 0 {
				t.Fatalf("un «no» no espera: %v", got.Wait)
			}
		})
	}
}

func TestLaEsperaPropiaCreceConTopeDe8YUnFactorEntre075Y1(t *testing.T) {
	esperados := map[int]float64{1: 0.5, 2: 1, 3: 2, 4: 4, 5: 8, 6: 8, 9: 8}
	for n, e := range esperados {
		if got := OwnWait(n, 1).Seconds(); math.Abs(got-e) > 1e-9 {
			t.Errorf("OwnWait(%d, 1) = %v, esperado %v", n, got, e)
		}
	}
	if got := OwnWait(1, 0).Seconds(); math.Abs(got-0.375) > 1e-9 {
		t.Errorf("OwnWait(1, 0) = %v", got)
	}
	if got := OwnWait(4, 0).Seconds(); math.Abs(got-3) > 1e-9 {
		t.Errorf("OwnWait(4, 0) = %v", got)
	}
}
