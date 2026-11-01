package trevol

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/trevol/trevol-go/generado"
)

type casoWebhook struct {
	Nombre    string            `json:"nombre"`
	Secreto   string            `json:"secreto"`
	Cuerpo    string            `json:"cuerpo"`
	Cabeceras map[string]string `json:"cabeceras"`
	Ahora     int64             `json:"ahora"`
	Valido    bool              `json:"valido"`
	Motivo    string            `json:"motivo"`
	Evento    struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"evento"`
}

func cargarWebhooks(t testing.TB) (tolerancia int, casos []casoWebhook) {
	var d struct {
		Tolerancia int           `json:"tolerancia"`
		Casos      []casoWebhook `json:"casos"`
	}
	datos(t, "webhooks.json", &d)
	return d.Tolerancia, d.Casos
}

func cabecerasDe(m map[string]string) http.Header {
	h := http.Header{}
	for k, v := range m {
		h.Set(k, v)
	}
	return h
}

func verificadorEn(t testing.TB, secreto string, ahora int64) *Webhook {
	w, err := NewWebhook(secreto)
	if err != nil {
		t.Fatal(err)
	}
	w.ahora = func() time.Time { return time.Unix(ahora, 0) }
	return w
}

func TestLaToleranciaEsDe5Minutos(t *testing.T) {
	tol, _ := cargarWebhooks(t)
	if tol != 300 || Tolerance != 5*time.Minute {
		t.Errorf("tolerancia = %d", tol)
	}
}

func TestVectoresDePruebasDatosWebhooksJSON(t *testing.T) {
	_, casos := cargarWebhooks(t)
	for _, c := range casos {
		t.Run(c.Nombre, func(t *testing.T) {
			w := verificadorEn(t, c.Secreto, c.Ahora)
			e, err := w.Verify([]byte(c.Cuerpo), cabecerasDe(c.Cabeceras))
			if c.Valido {
				if err != nil {
					t.Fatal(err)
				}
				if e.GetId() != c.Evento.ID || e.GetType() != c.Evento.Type {
					t.Errorf("evento = %s %s", e.GetId(), e.GetType())
				}
				return
			}
			var te *Error
			if !errors.As(err, &te) || te.Kind != KindWebhookVerification || te.Reason != c.Motivo {
				t.Fatalf("err = %#v, motivo esperado %s", err, c.Motivo)
			}
		})
	}
}

func okCaso(t testing.TB) casoWebhook {
	_, casos := cargarWebhooks(t)
	for _, c := range casos {
		if c.Nombre == "valido" {
			return c
		}
	}
	t.Fatal("no está el caso «valido»")
	return casoWebhook{}
}

func TestCabecerasDeCualquierCajaYValoresRepetidos(t *testing.T) {
	ok := okCaso(t)
	w := verificadorEn(t, ok.Secreto, ok.Ahora)
	h := http.Header{}
	h["webhook-id"] = []string{ok.Cabeceras["webhook-id"], "evt_otro"} // sin canonicalizar, como lo arma un http.Header a mano: Get no lo ve...
	h.Set("Webhook-Id", ok.Cabeceras["webhook-id"])
	h.Set("WEBHOOK-TIMESTAMP", ok.Cabeceras["webhook-timestamp"])
	h.Set("webhook-signature", ok.Cabeceras["webhook-signature"])
	if _, err := w.Verify([]byte(ok.Cuerpo), h); err != nil {
		t.Fatal(err)
	}
	// La primera cabecera repetida gana (Get).
	h2 := cabecerasDe(ok.Cabeceras)
	h2.Add("Webhook-Id", "evt_otro")
	if _, err := w.Verify([]byte(ok.Cuerpo), h2); err != nil {
		t.Fatal(err)
	}
}

func TestUnSecretoVacioOConBasuraEsErrorDeConfiguracionYNoSeFiltra(t *testing.T) {
	for _, malo := range []string{"", "   ", "whsec_"} {
		_, err := NewWebhook(malo)
		var te *Error
		if !errors.As(err, &te) || te.Kind != KindConfiguration {
			t.Errorf("%q: %v", malo, err)
		}
	}
	_, err := NewWebhook("whsec_@@@@@")
	if err == nil || strings.Contains(err.Error(), "@@@") {
		t.Errorf("err = %v", err)
	}
}

func TestEventDataDaElModeloSegunElTypeYUnTypeNuevoNoTieneModelo(t *testing.T) {
	ok := okCaso(t)
	w := verificadorEn(t, ok.Secreto, ok.Ahora)
	e, err := w.Verify([]byte(ok.Cuerpo), cabecerasDe(ok.Cabeceras))
	if err != nil {
		t.Fatal(err)
	}
	d, hay, err := EventData(e)
	if err != nil || !hay {
		t.Fatalf("%v %v", hay, err)
	}
	m, esModelo := d.(*generado.DocumentEventData)
	if !esModelo || m.GetStatus() != "accepted" {
		t.Errorf("data = %#v", d)
	}
	nuevo := "invoice.archived"
	e.Type = &nuevo
	if _, hay, _ := EventData(e); hay {
		t.Error("un type nuevo no tiene modelo")
	}
}

func TestElMensajeDeUnRechazoNoIncluyeElSecreto(t *testing.T) {
	ok := okCaso(t)
	w := verificadorEn(t, ok.Secreto, ok.Ahora)
	_, err := w.Verify([]byte(ok.Cuerpo+" "), cabecerasDe(ok.Cabeceras))
	if err == nil || strings.Contains(err.Error(), ok.Secreto[6:]) {
		t.Errorf("err = %v", err)
	}
}

func TestUnCuerpoQueNoEsUTF8SeRechaza(t *testing.T) {
	ok := okCaso(t)
	w := verificadorEn(t, ok.Secreto, ok.Ahora)
	_, err := w.Verify([]byte{0xff, 0xfe, 0xfd}, cabecerasDe(ok.Cabeceras))
	var te *Error
	if !errors.As(err, &te) || te.Reason != "cuerpo" {
		t.Errorf("err = %v", err)
	}
}

// ── Eventos del futuro ──
func firmar(t testing.TB, evento map[string]any) (string, []byte, http.Header, int64) {
	clave := []byte("0123456789abcdef0123456789abcdef")
	secreto := "whsec_" + base64.StdEncoding.EncodeToString(clave)
	ts := int64(1793491200)
	cuerpo, _ := json.Marshal(evento)
	m := hmac.New(sha256.New, clave)
	fmt.Fprintf(m, "%s.%d.%s", evento["id"], ts, cuerpo)
	h := http.Header{}
	h.Set("webhook-id", evento["id"].(string))
	h.Set("webhook-timestamp", fmt.Sprint(ts))
	h.Set("webhook-signature", "v1,"+base64.StdEncoding.EncodeToString(m.Sum(nil)))
	return secreto, cuerpo, h, ts
}
