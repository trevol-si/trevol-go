// Package trevol es el SDK de Go de la API de Trevol (facturación electrónica y demás documentos ante la DIAN, Colombia).
// Documentación: https://platform.trevol.si/docs
//
// El cliente generado (paquete generado) lleva una capa propia delgada: Idempotency-Key automática (la misma en los reintentos),
// reintentos con espera creciente que respetan Retry-After, errores tipados (*Error, con errors.As), entorno deducido del prefijo de la
// llave, Trevol-Version fijada, verificación de webhooks, paginación con iteradores y descargas de PDF/XML/ZIP.
package trevol

import (
	"context"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/trevol-si/trevol-go/generado"
)

// Version es la versión de este SDK; APIVersion la versión de la API con que se generó (el valor de Trevol-Version por defecto).
const (
	Version    = generado.Version
	APIVersion = generado.APIVersion
	// BaseURL es la URL de la API en producción y habilitación (la llave decide el entorno).
	BaseURL = "https://api.trevol.si/v1"
)

// Client es el cliente de la API: lleva embebido el cliente generado, así que cada recurso es un campo (c.InvoicesAPI, c.CreditNotesAPI,
// c.EventsAPI, c.WebhookEndpointsAPI, c.BalanceAPI, c.GeneralAPI…) con los métodos de la especificación:
//
//	factura, _, err := c.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).Execute()
type Client struct {
	*generado.APIClient
	// Environment es EnvTest (llave tvl_test_, habilitación de la DIAN) o EnvLive (tvl_live_, producción).
	Environment Environment
	// APIVersion es el valor de Trevol-Version que se manda en cada petición.
	APIVersion string
}

type config struct {
	apiVersion    string
	maxReintentos int
	timeout       time.Duration
	maxRetryAfter time.Duration
	baseURL       string
	entorno       Environment
	http          *http.Client
	sleep         func(context.Context, time.Duration) error
	random        func() float64
	uuid          func() string
	ahora         func() time.Time
}

// Option configura NewClient.
type Option func(*config)

// WithAPIVersion fija el valor de Trevol-Version (por defecto APIVersion).
func WithAPIVersion(v string) Option { return func(c *config) { c.apiVersion = v } }

// WithMaxRetries son los reintentos tras el primer intento (por defecto 2: 3 intentos en total). 0 los apaga.
func WithMaxRetries(n int) Option { return func(c *config) { c.maxReintentos = n } }

// WithTimeout es el tiempo máximo de cada intento (por defecto 65 s, como el playground).
func WithTimeout(d time.Duration) Option { return func(c *config) { c.timeout = d } }

// WithMaxRetryAfter: si Retry-After pide esperar más que esto (por defecto 60 s), no se espera: se devuelve el error con RetryAfter.
func WithMaxRetryAfter(d time.Duration) Option { return func(c *config) { c.maxRetryAfter = d } }

// WithBaseURL cambia la URL base (por defecto BaseURL), p. ej. para un servidor simulado en las pruebas.
func WithBaseURL(u string) Option { return func(c *config) { c.baseURL = u } }

// WithEnvironment exige un entorno: si la llave no coincide, NewClient devuelve un *Error KindConfiguration ANTES de llamar.
func WithEnvironment(e Environment) Option { return func(c *config) { c.entorno = e } }

// WithHTTPClient usa tu propio *http.Client (proxy, TLS, métricas…): su Transport es el que hace las peticiones y se envuelve con la
// capa de Trevol; el resto de sus campos (CheckRedirect, Jar) se respetan. Su Timeout, si lo pones, se suma al de cada intento.
func WithHTTPClient(h *http.Client) Option { return func(c *config) { c.http = h } }

// NewClient crea el cliente. Si apiKey está vacía se lee la variable de entorno TREVOL_API_KEY.
// Devuelve un *Error de clase KindConfiguration (sin red, y sin la llave en el mensaje) si la llave está mal formada o no coincide con el entorno pedido.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	c := config{apiVersion: APIVersion, maxReintentos: 2, timeout: 65 * time.Second, maxRetryAfter: 60 * time.Second, baseURL: BaseURL,
		sleep: dormir, random: rand.Float64, uuid: nuevoUUID, ahora: time.Now}
	for _, o := range opts {
		o(&c)
	}
	llave := strings.TrimSpace(apiKey)
	if llave == "" {
		llave = strings.TrimSpace(os.Getenv("TREVOL_API_KEY"))
	}
	if llave == "" {
		return nil, nuevoError(KindConfiguration, "Falta la llave de la API: pásala a trevol.NewClient(...) o define la variable de entorno TREVOL_API_KEY.")
	}
	entorno, err := environmentOf(llave)
	if err != nil {
		return nil, err
	}
	if c.entorno != "" && c.entorno != entorno {
		texto := "tvl_test_ = habilitación"
		if entorno == EnvLive {
			texto = "tvl_live_ = producción"
		}
		return nil, nuevoError(KindConfiguration, "Pediste el entorno «"+string(c.entorno)+"» pero la llave es de «"+string(entorno)+"» ("+texto+").")
	}
	if c.maxReintentos < 0 {
		c.maxReintentos = 0
	}

	base := c.http
	if base == nil {
		base = &http.Client{}
	}
	subyacente := base.Transport
	if subyacente == nil {
		subyacente = http.DefaultTransport
	}
	rutaBase := ""
	if i := strings.Index(c.baseURL, "://"); i >= 0 {
		if j := strings.Index(c.baseURL[i+3:], "/"); j >= 0 {
			rutaBase = strings.TrimRight(c.baseURL[i+3+j:], "/")
		}
	}
	cliente := *base // una copia: no se toca el *http.Client de la persona
	cliente.Transport = &transporte{base: subyacente, o: &opciones{
		llave: llave, apiVersion: c.apiVersion, userAgent: agenteDeUsuario(), maxReintentos: c.maxReintentos, timeout: c.timeout,
		maxRetryAfter: c.maxRetryAfter, rutaBase: rutaBase, sleep: c.sleep, random: c.random, uuid: c.uuid, ahora: c.ahora,
	}}

	cfg := generado.NewConfiguration()
	cfg.HTTPClient = &cliente
	cfg.UserAgent = agenteDeUsuario()
	cfg.Servers = generado.ServerConfigurations{{URL: strings.TrimRight(c.baseURL, "/"), Description: "Trevol"}}
	cfg.OperationServers = map[string]generado.ServerConfigurations{}
	return &Client{APIClient: generado.NewAPIClient(cfg), Environment: entorno, APIVersion: c.apiVersion}, nil
}
