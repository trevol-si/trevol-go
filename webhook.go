package trevol

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
	"github.com/trevol-si/trevol-go/generado"
)

// Tolerance es cuánto se tolera la diferencia entre webhook-timestamp y el reloj: 5 minutos hacia atrás y hacia adelante.
const Tolerance = 5 * time.Minute

// Webhook verifica las entregas de webhooks de Trevol (Standard Webhooks: webhook-id, webhook-timestamp, webhook-signature "v1,<b64>",
// secreto whsec_…). La firma se calcula sobre el CUERPO CRUDO: lee los bytes tal cual (io.ReadAll(r.Body)) antes de decodificar nada.
type Webhook struct {
	verificador *standardwebhooks.Webhook
	ahora       func() time.Time
}

// NewWebhook crea el verificador con el secreto del endpoint (whsec_…), que se muestra una sola vez al crearlo o rotarlo.
func NewWebhook(secret string) (*Webhook, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" || secret == "whsec_" {
		return nil, nuevoError(KindConfiguration, "Falta el secreto del webhook (whsec_…). Se muestra una sola vez al crear o rotar el endpoint.")
	}
	v, err := standardwebhooks.NewWebhook(secret)
	if err != nil { // base64 inválido: nunca se cuenta el secreto
		return nil, nuevoError(KindConfiguration, "El secreto del webhook no tiene el formato esperado (whsec_ + base64).")
	}
	return &Webhook{verificador: v, ahora: time.Now}, nil
}

func rechazo(motivo, mensaje string) *Error {
	e := nuevoError(KindWebhookVerification, mensaje)
	e.Reason = motivo
	return e
}

// Verify verifica la firma y la fecha de una entrega y devuelve el evento. Si algo no cuadra devuelve un *Error de clase
// KindWebhookVerification cuyo Reason es el motivo: "firma", "fecha", "cabeceras" o "cuerpo".
//
// Ojo: la verificación NO evita repeticiones dentro de los 5 minutos ni los reintentos de entrega: deduplica por `webhook-id`.
func (w *Webhook) Verify(body []byte, header http.Header) (*generado.Event, error) {
	if !utf8.Valid(body) {
		return nil, rechazo("cuerpo", "El cuerpo del webhook no es UTF-8.")
	}
	for _, n := range []string{standardwebhooks.HeaderWebhookID, standardwebhooks.HeaderWebhookTimestamp, standardwebhooks.HeaderWebhookSignature} {
		if strings.TrimSpace(header.Get(n)) == "" {
			return nil, rechazo("cabeceras", "Falta la cabecera "+n+".")
		}
	}
	marca := header.Get(standardwebhooks.HeaderWebhookTimestamp)
	if !soloDigitos(marca) {
		return nil, rechazo("fecha", "El webhook-timestamp no es un número de segundos.")
	}
	segundos, err := strconv.ParseInt(marca, 10, 64)
	if err != nil {
		return nil, rechazo("fecha", "El webhook-timestamp no es un número de segundos.")
	}
	// La tolerancia la fija este SDK (con el reloj inyectable en las pruebas); la biblioteca de Standard Webhooks verifica la firma.
	desfase := w.ahora().Sub(time.Unix(segundos, 0))
	if desfase > Tolerance || desfase < -Tolerance {
		return nil, rechazo("fecha", "La fecha del webhook está fuera de la tolerancia de 5 minutos.")
	}
	if err := w.verificador.VerifyIgnoringTimestamp(body, header); err != nil {
		return nil, rechazo("firma", "La firma del webhook no coincide: revisa el secreto y que el cuerpo sea el crudo, sin reescribir.")
	}
	var evento generado.Event
	if err := json.Unmarshal(body, &evento); err != nil {
		return nil, rechazo("cuerpo", "El cuerpo firmado no es un evento válido.")
	}
	return &evento, nil
}

func soloDigitos(s string) bool {
	if len(s) == 0 || len(s) > 12 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// EventData da el `data` del evento como el modelo de su `type` (p. ej. *generado.DocumentEventData para invoice.accepted).
// ok es false si el tipo es nuevo y este SDK todavía no lo conoce: el evento no se pierde, evento.Data sigue ahí como mapa.
func EventData(evento *generado.Event) (data any, ok bool, err error) {
	if evento == nil || evento.Type == nil || evento.Data == nil {
		return nil, false, nil
	}
	decodificar, existe := generado.EventoDatos[*evento.Type]
	if !existe {
		return nil, false, nil
	}
	crudo, err := json.Marshal(evento.Data)
	if err != nil {
		return nil, false, err
	}
	d, err := decodificar(crudo)
	return d, err == nil, err
}
