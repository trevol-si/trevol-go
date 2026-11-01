package trevol

import (
	"context"
	"io"
	"math"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/trevol/trevol-go/generado"
)

// Outcome es lo que pasó en un intento: una respuesta (con Status) o un fallo de red (Network, sin respuesta).
type Outcome struct {
	Network   bool
	Status    int
	Code      string
	Retryable *bool
	// RetryAfter ya resuelto (la cabecera, o `retry_after` del cuerpo).
	RetryAfter time.Duration
}

// Decision es la respuesta de Decide: si se reintenta y cuánto se espera.
type Decision struct {
	Retry bool
	// Wait es la espera antes del siguiente intento.
	Wait time.Duration
	// Reason explica un «no»: "exito", "agotado" o "retry_after_excede" ("" si es un error final).
	Reason string
}

var estadosReintentables = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}

// OwnWait es la espera propia antes del intento siguiente: 0,5 · 2^(intento−1), con tope de 8 s, por un factor al azar en [0,75; 1].
// `random` está en [0, 1).
func OwnWait(attempt int, random float64) time.Duration {
	base := math.Min(0.5*math.Pow(2, float64(attempt-1)), 8)
	return time.Duration(base * (0.75 + 0.25*random) * float64(time.Second))
}

// Decide es una función PURA: ¿se reintenta y cuánto se espera? Las mismas reglas y los mismos casos (pruebas/datos/transporte.json)
// en los 7 SDK. `class` es "seguro" (GET, preview, PATCH y todo POST/PUT con Idempotency-Key) o "solo_429" (el resto).
func Decide(attempt int, class string, maxRetries int, maxRetryAfter time.Duration, random float64, o Outcome) Decision {
	if !o.Network && o.Status >= 200 && o.Status < 300 {
		return Decision{Reason: "exito"}
	}
	var reintentable bool
	switch {
	case o.Network:
		reintentable = class == "seguro"
	case class == "solo_429":
		reintentable = o.Status == 429
	case o.Status == 402:
		reintentable = false
	default:
		reintentable = estadosReintentables[o.Status] || o.Code == "request_in_progress" || o.Code == "previous_event_pending" ||
			(o.Retryable != nil && *o.Retryable)
	}
	if !reintentable {
		return Decision{}
	}
	if attempt > maxRetries {
		return Decision{Reason: "agotado"}
	}
	pedida := time.Duration(0)
	if !o.Network && o.RetryAfter > 0 {
		pedida = o.RetryAfter
	}
	if pedida > maxRetryAfter {
		return Decision{Reason: "retry_after_excede"}
	}
	espera := OwnWait(attempt, random)
	if pedida > espera {
		espera = pedida
	}
	return Decision{Retry: true, Wait: espera}
}

var reClave = regexp.MustCompile(`^[!-~]{1,255}$`)

// ── Qué operación es ────────────────────────────────────────────────────────

type plantilla struct {
	op generado.Operacion
	re *regexp.Regexp
}

var (
	opsExactas    map[string]generado.Operacion
	opsPlantillas []plantilla
	opsUnaVez     sync.Once
)

func cargarOperaciones() {
	opsUnaVez.Do(func() {
		opsExactas = map[string]generado.Operacion{}
		for _, o := range generado.Operaciones {
			if !strings.Contains(o.Ruta, "{") {
				opsExactas[o.Metodo+" "+o.Ruta] = o
				continue
			}
			re := regexp.MustCompile("^" + regexp.MustCompile(`\\\{[a-z_]+\\\}`).ReplaceAllString(regexp.QuoteMeta(o.Ruta), `[^/]+`) + "$")
			opsPlantillas = append(opsPlantillas, plantilla{o, re})
		}
	})
}

// operacionDe busca la operación por método y ruta (sin el prefijo /v1). Las rutas fijas ganan a las plantillas
// (GET /radian_events/tacit_acceptance frente a GET /radian_events/{id}).
func operacionDe(metodo, ruta string) (generado.Operacion, bool) {
	cargarOperaciones()
	if o, ok := opsExactas[metodo+" "+ruta]; ok {
		return o, true
	}
	for _, p := range opsPlantillas {
		if p.op.Metodo == metodo && p.re.MatchString(ruta) {
			return p.op, true
		}
	}
	return generado.Operacion{}, false
}

const problema = "application/problem+json"

func aceptar(binario string) string {
	switch binario {
	case "pdf":
		return "application/pdf, " + problema
	case "xml":
		return "application/xml, " + problema
	case "zip":
		return "application/zip, " + problema
	}
	return "application/json, " + problema
}

func generadoClasePorCode(code string) string { return generado.ClasePorCode[code] }

func generadoClasePorEstado(estado int) string {
	if c, ok := generado.ClasePorEstado[strconv.Itoa(estado)]; ok {
		return c
	}
	if estado >= 500 {
		return generado.ClasePorEstado["5xx"]
	}
	return ""
}

// ── El http.RoundTripper ────────────────────────────────────────────────────

type opciones struct {
	llave         string
	apiVersion    string
	userAgent     string
	maxReintentos int
	timeout       time.Duration
	maxRetryAfter time.Duration
	rutaBase      string
	sleep         func(context.Context, time.Duration) error
	random        func() float64
	uuid          func() string
	ahora         func() time.Time
}

// transporte es el http.RoundTripper de Trevol: pone las cabeceras y la Idempotency-Key, reintenta y convierte toda respuesta que no
// es 2xx en un *Error que atraviesa el código generado sin envolverse (el generado devuelve el error de `HTTPClient.Do` tal cual; net/http
// lo envuelve en *url.Error y errors.As lo saca).
type transporte struct {
	base http.RoundTripper
	o    *opciones
}

func dormir(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type cancelaAlCerrar struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c cancelaAlCerrar) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

func (t *transporte) RoundTrip(req *http.Request) (*http.Response, error) {
	o := t.o
	ruta := strings.TrimPrefix(req.URL.Path, o.rutaBase)
	op, conocida := operacionDe(req.Method, ruta)

	clase := "solo_429"
	if req.Method == http.MethodGet {
		clase = "seguro"
	}
	if conocida {
		clase = op.Reintento
	}

	// La Idempotency-Key: una por llamada, la misma en todos los reintentos.
	clave := ""
	cabeceras := req.Header.Clone()
	if cabeceras == nil {
		cabeceras = http.Header{}
	}
	if conocida && op.Idempotente {
		if dada := cabeceras.Get("Idempotency-Key"); dada != "" || len(cabeceras.Values("Idempotency-Key")) > 0 {
			if !reClave.MatchString(dada) {
				return nil, nuevoError(KindConfiguration, "La idempotency key debe tener de 1 a 255 caracteres ASCII visibles (sin espacios).")
			}
			clave = dada
		} else {
			clave = o.uuid()
		}
		cabeceras.Set("Idempotency-Key", clave)
	} else {
		cabeceras.Del("Idempotency-Key")
	}
	cabeceras.Set("Authorization", "Bearer "+o.llave)
	cabeceras.Set("Trevol-Version", o.apiVersion)
	cabeceras.Set("User-Agent", o.userAgent)
	binario := ""
	if conocida {
		binario = op.Binario
	}
	cabeceras.Set("Accept", aceptar(binario))

	var ultimo *Error
	for intento := 1; ; intento++ {
		if intento > 1 && req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
			return nil, ultimo // no se puede rebobinar el cuerpo: no se reintenta a ciegas
		}
		resp, fallo, decision, errPropio := t.intentar(req, cabeceras, intento, clase, clave)
		if errPropio != nil {
			return nil, errPropio
		}
		if resp != nil {
			return resp, nil
		}
		ultimo = fallo
		if !decision.Retry {
			return nil, ultimo
		}
		if errDormir := o.sleep(req.Context(), decision.Wait); errDormir != nil {
			return nil, errDormir
		}
	}
}

// intentar hace UN intento. Devuelve la respuesta si fue 2xx; si no, el *Error de ese intento y la decisión de reintentar. errPropio es un
// error que se devuelve tal cual (la persona canceló su contexto, o no se pudo rebobinar el cuerpo).
func (t *transporte) intentar(req *http.Request, cabeceras http.Header, intento int, clase, clave string) (*http.Response, *Error, Decision, error) {
	o := t.o
	ctx, cancelar := context.WithTimeout(req.Context(), o.timeout)
	r := req.Clone(ctx)
	r.Header = cabeceras.Clone()
	if intento > 1 && req.Body != nil && req.Body != http.NoBody {
		cuerpo, err := req.GetBody()
		if err != nil {
			cancelar()
			return nil, nil, Decision{}, err
		}
		r.Body = cuerpo
	}
	resp, err := t.base.RoundTrip(r)
	if err != nil {
		cancelar()
		if req.Context().Err() != nil { // la persona canceló (o venció su contexto): no es un fallo de red
			return nil, nil, Decision{}, err
		}
		fallo := &Error{Kind: KindConnection, IdempotencyKey: clave, causa: err, mensaje: "No pude conectar con la API de Trevol (" + resumen(err) + ")."}
		return nil, fallo, Decide(intento, clase, o.maxReintentos, o.maxRetryAfter, o.random(), Outcome{Network: true}), nil
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		resp.Body = cancelaAlCerrar{resp.Body, cancelar}
		return resp, nil, Decision{}, nil
	}
	cuerpo, errLeer := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	cancelar()
	if errLeer != nil {
		fallo := &Error{Kind: KindConnection, IdempotencyKey: clave, causa: errLeer, mensaje: "No pude leer la respuesta de la API de Trevol."}
		return nil, fallo, Decide(intento, clase, o.maxReintentos, o.maxRetryAfter, o.random(), Outcome{Network: true}), nil
	}
	fallo := errorDeRespuesta(resp.StatusCode, cuerpo, resp.Header, clave, o.ahora())
	var retryable *bool
	if v, ok := fallo.Problem["retryable"].(bool); ok {
		retryable = &v
	}
	return nil, fallo, Decide(intento, clase, o.maxReintentos, o.maxRetryAfter, o.random(),
		Outcome{Status: resp.StatusCode, Code: fallo.Code, Retryable: retryable, RetryAfter: fallo.RetryAfter}), nil
}

func resumen(err error) string {
	texto := err.Error()
	if len(texto) > 200 {
		texto = texto[:200]
	}
	return texto
}

func agenteDeUsuario() string {
	return "trevol-go/" + generado.Version + " (" + runtime.Version() + "; +https://platform.trevol.si/docs)"
}
