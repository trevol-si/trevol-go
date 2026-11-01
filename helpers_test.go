package trevol

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	llaveTest = "tvl_test_a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
	llaveLive = "tvl_live_b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2"
	hex32     = "3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
)

var (
	idDoc     = "inv_" + hex32
	documento = map[string]any{"id": idDoc, "object": "invoice", "environment": "test", "status": "accepted", "number": "SETP990012345", "external_id": "VENTA-2026-000123"}
)

func datos(t testing.TB, nombre string, destino any) {
	t.Helper()
	crudo, err := os.ReadFile(filepath.Join("pruebas", "datos", nombre))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(crudo, destino); err != nil {
		t.Fatal(err)
	}
}

// resp es una respuesta del servidor falso.
type resp struct {
	estado    int
	cuerpo    any
	crudo     []byte
	tipo      string
	cabeceras map[string]string
	cortar    bool // el servidor cierra la conexión sin responder (un fallo de red)
}

func jsonResp(estado int, cuerpo any) resp {
	return resp{estado: estado, cuerpo: cuerpo, tipo: "application/json"}
}

func problemaResp(estado int, code string, extra map[string]any, cabeceras map[string]string) resp {
	cuerpo := map[string]any{"type": "https://platform.trevol.si/docs/errores#" + code, "title": "Título", "status": estado, "code": code,
		"retryable": false, "detail": "Detalle de " + code, "request_id": "req_c3c3c3c3c3c3c3c3c3c3c3c3"}
	for k, v := range extra {
		cuerpo[k] = v
	}
	return resp{estado: estado, cuerpo: cuerpo, tipo: "application/problem+json", cabeceras: cabeceras}
}

type peticion struct {
	metodo, ruta string
	cabeceras    http.Header
	cuerpo       string
}

// banco es un cliente contra un servidor HTTP falso (pasa por net/http de verdad), con el reloj y el azar inyectados.
type banco struct {
	t          *testing.T
	srv        *httptest.Server
	mu         sync.Mutex
	respuestas []any // resp o func(peticion) resp
	pet        []peticion
	esperas    []time.Duration
	nUUID      int
	cliente    *Client
}

func conInternos(b *banco) Option {
	return func(c *config) {
		c.sleep = func(_ context.Context, d time.Duration) error { b.esperas = append(b.esperas, d); return nil }
		c.random = func() float64 { return 1 }
		c.uuid = func() string {
			b.nUUID++
			return "00000000-0000-4000-8000-" + strings.Repeat("0", 11) + string(rune('0'+b.nUUID))
		}
	}
}

func nuevoBanco(t *testing.T, respuestas ...any) *banco {
	return nuevoBancoConLlave(t, llaveTest, respuestas, nil)
}

func nuevoBancoConLlave(t *testing.T, llave string, respuestas []any, opts []Option) *banco {
	t.Helper()
	b := &banco{t: t, respuestas: respuestas}
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		b.mu.Lock()
		p := peticion{metodo: r.Method, ruta: r.URL.RequestURI(), cabeceras: r.Header.Clone(), cuerpo: string(cuerpo)}
		b.pet = append(b.pet, p)
		i := len(b.pet) - 1
		if i >= len(b.respuestas) {
			i = len(b.respuestas) - 1
		}
		x := b.respuestas[i]
		b.mu.Unlock()
		var rr resp
		switch v := x.(type) {
		case resp:
			rr = v
		case func(peticion) resp:
			rr = v(p)
		}
		if rr.cortar {
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		crudo := rr.crudo
		if crudo == nil && rr.cuerpo != nil {
			crudo, _ = json.Marshal(rr.cuerpo)
		}
		w.Header().Set("Content-Type", rr.tipo)
		for k, v := range rr.cabeceras {
			w.Header().Set(k, v)
		}
		w.WriteHeader(rr.estado)
		w.Write(crudo)
	}))
	t.Cleanup(b.srv.Close)
	todas := append([]Option{WithBaseURL(b.srv.URL + "/v1"), conInternos(b)}, opts...)
	c, err := NewClient(llave, todas...)
	if err != nil {
		t.Fatal(err)
	}
	b.cliente = c
	return b
}

func (b *banco) peticiones() []peticion {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]peticion(nil), b.pet...)
}
