// Package trevol es el SDK de Go de la API de Trevol (facturación electrónica y demás documentos ante la DIAN, Colombia).
// Documentación: https://platform.trevol.si/docs
//
// El cliente generado (paquete generado) lleva una capa propia delgada: Idempotency-Key automática (la misma en los reintentos),
// reintentos con espera creciente que respetan Retry-After, errores tipados (*Error, con errors.As), entorno deducido del prefijo de la
// llave, Trevol-Version fijada, la llave de administración (tva_…) con Trevol-Company y Trevol-Environment, verificación de webhooks,
// paginación con iteradores y descargas de PDF/XML/ZIP.
package trevol

import (
	"context"
	"fmt"
	"maps"
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
// c.EventsAPI, c.WebhookEndpointsAPI, c.BalanceAPI, c.GeneralAPI, c.AccountAPI…) con los métodos de la especificación:
//
//	factura, _, err := c.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).Execute()
//
// Al imprimirlo (%v, %+v, %#v) nunca sale la llave: solo su prefijo.
type Client struct {
	*generado.APIClient
	// Environment es EnvTest (llave tvl_test_ o tva_test_, habilitación de la DIAN) o EnvLive (tvl_live_ o tva_live_, producción).
	Environment Environment
	// KeyType es KeyTypeAPI (tvl_…, la llave de una empresa) o KeyTypeAccount (tva_…, la llave de administración).
	KeyType KeyType
	// APIVersion es el valor de Trevol-Version que se manda en cada petición.
	APIVersion string

	t *transporte
}

// Company es la empresa (com_…) a la que este cliente manda las peticiones de empresa con una llave de administración ("" si no hay).
func (c *Client) Company() string {
	if c == nil || c.t == nil {
		return ""
	}
	return c.t.o.empresa
}

// String describe el cliente sin la llave (solo su prefijo).
func (c *Client) String() string {
	if c == nil || c.t == nil {
		return "trevol.Client{}"
	}
	empresa := c.t.o.empresa
	if empresa == "" {
		empresa = "-"
	}
	return fmt.Sprintf("trevol.Client{key: %s, environment: %s, key_type: %s, company: %s, api_version: %s}",
		llaveVisible(c.t.o.llave), c.Environment, c.KeyType, empresa, c.APIVersion)
}

// Format hace que todo verbo de fmt (%v, %+v, %#v, %d…) muestre String: ninguno recorre los campos internos.
func (c *Client) Format(f fmt.State, _ rune) { fmt.Fprint(f, c.String()) }

// ForCompany devuelve un cliente para otra empresa de la cuenta: comparte el transporte, las conexiones y las opciones con c y solo cambia
// Trevol-Company. Es barato: créalo por petición si hace falta. Solo con una llave de administración (tva_…); con una tvl_, o con un id mal
// formado, devuelve un *Error KindConfiguration sin llamar.
//
//	ventas, err := c.ForCompany("com_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b")
func (c *Client) ForCompany(id string) (*Client, error) {
	if c == nil || c.t == nil {
		return nil, nuevoError(KindConfiguration, "Crea el cliente con trevol.NewClient antes de llamar a ForCompany.")
	}
	empresa, err := validarEmpresa(c.KeyType, id)
	if err != nil {
		return nil, err
	}
	o := *c.t.o
	o.empresa = empresa
	t := &transporte{base: c.t.base, o: &o}

	cfg := *c.GetConfig()
	cfg.DefaultHeader = maps.Clone(cfg.DefaultHeader)
	if cfg.DefaultHeader == nil {
		cfg.DefaultHeader = map[string]string{}
	}
	h := *cfg.HTTPClient
	h.Transport = t
	cfg.HTTPClient = &h
	return &Client{APIClient: generado.NewAPIClient(&cfg), Environment: c.Environment, KeyType: c.KeyType, APIVersion: c.APIVersion, t: t}, nil
}

type config struct {
	apiVersion    string
	maxReintentos int
	timeout       time.Duration
	maxRetryAfter time.Duration
	baseURL       string
	entorno       Environment
	empresa       string
	conEmpresa    bool
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

// WithCompany fija la empresa (com_…) de una llave de administración (tva_…): el SDK manda Trevol-Company con ese id y
// Trevol-Environment con el entorno de la llave en cada petición a una ruta de empresa. Con una llave tvl_, o con un id mal formado,
// NewClient devuelve un *Error KindConfiguration. Para otra empresa con las mismas conexiones: c.ForCompany(id).
func WithCompany(id string) Option { return func(c *config) { c.empresa, c.conEmpresa = id, true } }

// WithEnvironment exige un entorno: si la llave no coincide, NewClient devuelve un *Error KindConfiguration ANTES de llamar.
func WithEnvironment(e Environment) Option { return func(c *config) { c.entorno = e } }

// WithHTTPClient usa tu propio *http.Client (proxy, TLS, métricas…): su Transport es el que hace las peticiones y se envuelve con la
// capa de Trevol; el resto de sus campos (CheckRedirect, Jar) se respetan. Su Timeout, si lo pones, se suma al de cada intento.
func WithHTTPClient(h *http.Client) Option { return func(c *config) { c.http = h } }

// NewClient crea el cliente. Si apiKey está vacía se lee la variable de entorno TREVOL_API_KEY.
// La llave puede ser de una empresa (tvl_test_…, tvl_live_…) o de administración (tva_test_…, tva_live_…); el entorno sale del prefijo.
// Devuelve un *Error de clase KindConfiguration (sin red, y sin la llave en el mensaje) si la llave está mal formada, no coincide con el
// entorno pedido o WithCompany no aplica.
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
	entorno, clase, err := analizarLlave(llave)
	if err != nil {
		return nil, err
	}
	if c.entorno != "" && c.entorno != entorno {
		texto := llaveVisible(llave) + " = habilitación"
		if entorno == EnvLive {
			texto = llaveVisible(llave) + " = producción"
		}
		return nil, nuevoError(KindConfiguration, "Pediste el entorno «"+string(c.entorno)+"» pero la llave es de «"+string(entorno)+"» ("+texto+").")
	}
	empresa := ""
	if c.conEmpresa {
		if empresa, err = validarEmpresa(clase, c.empresa); err != nil {
			return nil, err
		}
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
	t := &transporte{base: subyacente, o: &opciones{
		llave: llave, clase: clase, entorno: entorno, empresa: empresa, apiVersion: c.apiVersion, userAgent: agenteDeUsuario(),
		maxReintentos: c.maxReintentos, timeout: c.timeout, maxRetryAfter: c.maxRetryAfter, rutaBase: rutaBase, sleep: c.sleep, random: c.random, uuid: c.uuid, ahora: c.ahora,
	}}
	cliente.Transport = t

	cfg := generado.NewConfiguration()
	cfg.HTTPClient = &cliente
	cfg.UserAgent = agenteDeUsuario()
	cfg.Servers = generado.ServerConfigurations{{URL: strings.TrimRight(c.baseURL, "/"), Description: "Trevol"}}
	cfg.OperationServers = map[string]generado.ServerConfigurations{}
	return &Client{APIClient: generado.NewAPIClient(cfg), Environment: entorno, KeyType: clase, APIVersion: c.apiVersion, t: t}, nil
}
