package trevol

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Kind es la clase de un *Error (docs/api/43 §2.6.5). Sale del `code` del problem+json: el mismo vocabulario que la documentación
// (https://platform.trevol.si/docs/errores).
type Kind string

const (
	// KindUnknown es la base: un `code` que este SDK todavía no conoce (la API sumó uno). Nunca rompe.
	KindUnknown Kind = "TrevolError"
	// KindAuthentication: authentication_required, invalid_api_key.
	KindAuthentication Kind = "AuthenticationError"
	// KindPermissionDenied: permission_denied.
	KindPermissionDenied Kind = "PermissionDeniedError"
	// KindNotFound: not_found.
	KindNotFound Kind = "NotFoundError"
	// KindInvalidRequest: method_not_allowed, unsupported_version, invalid_json, invalid_request, idempotency_key_required.
	KindInvalidRequest Kind = "InvalidRequestError"
	// KindIdempotency: idempotency_key_reused.
	KindIdempotency Kind = "IdempotencyError"
	// KindValidation: validation_error (422). `Errors` dice qué campo (`Pointer`) y por qué.
	KindValidation Kind = "ValidationError"
	// KindConflict: external_id_in_use, invalid_state, duplicate_document, request_in_progress, previous_event_pending,
	// file_unavailable, webhook_limit_reached.
	KindConflict Kind = "ConflictError"
	// KindPrecondition: company_not_enabled, numbering_unavailable, certificate_problem, tacit_acceptance_not_due.
	KindPrecondition Kind = "PreconditionError"
	// KindDianRejected: dian_rejected (`Problem["document"]` y `Problem["rules"]`).
	KindDianRejected Kind = "DianRejectedError"
	// KindBalance: insufficient_balance, top_up_below_minimum, top_up_above_maximum, top_ups_unavailable.
	KindBalance Kind = "BalanceError"
	// KindRateLimit: rate_limited.
	KindRateLimit Kind = "RateLimitError"
	// KindDianUnavailable: dian_unavailable.
	KindDianUnavailable Kind = "DianUnavailableError"
	// KindInternalServer: internal_error y cualquier 5xx sin `code` conocido.
	KindInternalServer Kind = "InternalServerError"
	// KindConnection: no hubo respuesta (red, DNS, TLS, tiempo agotado). Lleva la IdempotencyKey para reintentar sin duplicar.
	KindConnection Kind = "APIConnectionError"
	// KindConfiguration: mal uso del SDK, antes de llamar (llave mal formada, entorno que no coincide). Nunca incluye la llave.
	KindConfiguration Kind = "ConfigurationError"
	// KindWebhookVerification: la firma, la fecha o las cabeceras de un webhook no son válidas.
	KindWebhookVerification Kind = "WebhookVerificationError"
)

var kindPorClase = map[string]Kind{
	"AuthenticationError":   KindAuthentication,
	"PermissionDeniedError": KindPermissionDenied,
	"NotFoundError":         KindNotFound,
	"InvalidRequestError":   KindInvalidRequest,
	"IdempotencyError":      KindIdempotency,
	"ValidationError":       KindValidation,
	"ConflictError":         KindConflict,
	"PreconditionError":     KindPrecondition,
	"DianRejectedError":     KindDianRejected,
	"BalanceError":          KindBalance,
	"RateLimitError":        KindRateLimit,
	"DianUnavailableError":  KindDianUnavailable,
	"InternalServerError":   KindInternalServer,
}

// FieldError es una de las `errors[]` de un problema 422: qué campo falló (`Pointer`) y por qué.
type FieldError struct {
	Field   string
	Pointer string
	Message string
	// Extra son los campos que el SDK no conoce todavía.
	Extra map[string]any
}

// Error es el único tipo de error del SDK. Se saca con errors.As, también a través del *url.Error con que net/http lo envuelve:
//
//	var te *trevol.Error
//	if errors.As(err, &te) && te.Kind == trevol.KindValidation { ... }
//
// Ningún campo ni mensaje incluye la llave de la API ni las cabeceras de la PETICIÓN.
type Error struct {
	Kind      Kind
	Status    int
	Code      string
	Type      string
	Title     string
	Detail    string
	RequestID string
	// Retryable es lo que dice el problema (false si no lo dice).
	Retryable bool
	// RetryAfter es la espera que pidió la API (0 si no pidió ninguna).
	RetryAfter time.Duration
	Errors     []FieldError
	// IdempotencyKey es la clave de la llamada: úsala para reintentar más tarde con la MISMA clave.
	IdempotencyKey string
	// Problem es el cuerpo problem+json completo, con los campos extra (existing_id, reason, document, rules...).
	Problem map[string]any
	// Header son las cabeceras de la RESPUESTA.
	Header http.Header
	// Reason es el motivo de un KindWebhookVerification: "firma", "fecha", "cabeceras" o "cuerpo".
	Reason string

	mensaje string
	causa   error
}

// Error implementa error: clase, code, estado, detalle y request_id; nunca la llave.
func (e *Error) Error() string {
	partes := []string{string(e.Kind)}
	if e.Code != "" {
		partes = append(partes, e.Code)
	}
	if e.Status != 0 {
		partes = append(partes, "HTTP "+strconv.Itoa(e.Status))
	}
	texto := e.Detail
	if texto == "" {
		texto = e.Title
	}
	if texto == "" {
		texto = e.mensaje
	}
	cola := ""
	if e.RequestID != "" {
		cola = " [request_id=" + e.RequestID + "]"
	}
	return strings.Join(partes, " ") + ": " + texto + cola
}

// Unwrap devuelve la causa (el fallo de red de un KindConnection).
func (e *Error) Unwrap() error { return e.causa }

// Is permite errors.Is(err, &trevol.Error{Kind: trevol.KindNotFound}): compara la clase (y el code si el objetivo lo trae).
func (e *Error) Is(objetivo error) bool {
	o, ok := objetivo.(*Error)
	if !ok || o.Kind == "" {
		return false
	}
	return e.Kind == o.Kind && (o.Code == "" || e.Code == o.Code)
}

func nuevoError(kind Kind, mensaje string) *Error { return &Error{Kind: kind, mensaje: mensaje} }

const recorte = 2048

func numero(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, n >= 0 && n == n
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil && f >= 0 && f == f && f < 1e12
	}
	return 0, false
}

// leerRetryAfter lee el valor de `Retry-After`: segundos o una fecha HTTP. ok=false si no está o no se entiende.
func leerRetryAfter(valor string, ahora time.Time) (time.Duration, bool) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return 0, false
	}
	if n, ok := numero(valor); ok {
		return time.Duration(n * float64(time.Second)), true
	}
	if f, err := http.ParseTime(valor); err == nil {
		d := f.Sub(ahora)
		if d < 0 {
			d = 0
		}
		return d.Truncate(time.Second) + time.Second*boolSeg(d%time.Second != 0), true
	}
	return 0, false
}

func boolSeg(b bool) time.Duration {
	if b {
		return 1
	}
	return 0
}

// errorDeRespuesta convierte una respuesta que no es 2xx en su *Error. El cuerpo se lee como un mapa TOLERANTE (nunca con el modelo
// generado: un `code` nuevo o un campo que falta no puede romper el manejo de errores).
func errorDeRespuesta(estado int, cuerpo []byte, cab http.Header, idempotencyKey string, ahora time.Time) *Error {
	var mapa map[string]any
	esJSON := json.Unmarshal(cuerpo, &mapa) == nil && mapa != nil
	if !esJSON {
		mapa = nil
	}
	cadena := func(clave string) string {
		if s, ok := mapa[clave].(string); ok {
			return s
		}
		return ""
	}
	code, detail, title := cadena("code"), cadena("detail"), cadena("title")
	e := &Error{Status: estado, Code: code, Type: cadena("type"), Title: title, IdempotencyKey: idempotencyKey, Problem: mapa, Header: cab.Clone()}
	if lista, ok := mapa["errors"].([]any); ok {
		for _, x := range lista {
			m, ok := x.(map[string]any)
			if !ok {
				continue
			}
			fe := FieldError{Extra: map[string]any{}}
			fe.Field, _ = m["field"].(string)
			fe.Pointer, _ = m["pointer"].(string)
			fe.Message, _ = m["message"].(string)
			for k, v := range m {
				if k != "field" && k != "pointer" && k != "message" {
					fe.Extra[k] = v
				}
			}
			e.Errors = append(e.Errors, fe)
		}
	}
	if r, ok := mapa["retryable"].(bool); ok {
		e.Retryable = r
	}
	if d, ok := leerRetryAfter(cab.Get("Retry-After"), ahora); ok {
		e.RetryAfter = d
	} else if n, ok := numero(mapa["retry_after"]); ok {
		e.RetryAfter = time.Duration(n * float64(time.Second))
	}
	e.RequestID = cadena("request_id")
	if e.RequestID == "" {
		e.RequestID = cab.Get("Request-Id")
	}
	switch {
	case esJSON:
		e.Detail = detail
		e.mensaje = firstNonEmpty(detail, title, fmt.Sprintf("La API respondió %d", estado))
	default:
		texto := strings.TrimSpace(string(cuerpo))
		if len(texto) > recorte {
			texto = texto[:recorte]
		}
		e.Detail = texto
		e.mensaje = fmt.Sprintf("La API respondió %d sin problem+json", estado)
	}
	var clase string
	if code != "" {
		clase = generadoClasePorCode(code) // un code que el SDK no conoce (la API sumó uno) da la base, nunca rompe
	} else {
		clase = generadoClasePorEstado(estado)
	}
	if k, ok := kindPorClase[clase]; ok {
		e.Kind = k
	} else {
		e.Kind = KindUnknown
	}
	return e
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
