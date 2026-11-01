package trevol

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/trevol-si/trevol-go/generado"
)

var entradaFactura = generado.InvoiceInput{}

func crearFactura(t *testing.T, b *banco, claves ...string) (*generado.Document, error) {
	t.Helper()
	var entrada generado.InvoiceInput
	if err := jsonDe(`{"external_id":"VENTA-2026-000123","customer":{"final_consumer":true},"items":[{"description":"Café","quantity":"1","unit_price":"1000"}]}`, &entrada); err != nil {
		t.Fatal(err)
	}
	r := b.cliente.InvoicesAPI.CreateInvoice(context.Background()).InvoiceInput(entrada)
	if len(claves) > 0 {
		r = r.IdempotencyKey(claves[0])
	}
	d, _, err := r.Execute()
	return d, err
}

func TestAuthorizationTrevolVersionFijaUserAgentYNingunaXTrevol(t *testing.T) {
	b := nuevoBanco(t, jsonResp(200, documento))
	if _, _, err := b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute(); err != nil {
		t.Fatal(err)
	}
	h := b.peticiones()[0].cabeceras
	if h.Get("Authorization") != "Bearer "+llaveTest {
		t.Errorf("Authorization = %q", h.Get("Authorization"))
	}
	if h.Get("Trevol-Version") != "2026-10-22" || APIVersion != "2026-10-22" {
		t.Errorf("Trevol-Version = %q", h.Get("Trevol-Version"))
	}
	if !regexp.MustCompile(`^trevol-go/` + regexp.QuoteMeta(Version) + ` \(go[\d.]+[^;]*; \+https://platform\.trevol\.si/docs\)$`).MatchString(h.Get("User-Agent")) {
		t.Errorf("User-Agent = %q", h.Get("User-Agent"))
	}
	if h.Get("Accept") != "application/json, application/problem+json" {
		t.Errorf("Accept = %q", h.Get("Accept"))
	}
	for k := range h {
		if strings.HasPrefix(strings.ToLower(k), "x-trevol-") || strings.EqualFold(k, "trevol-empresa") || strings.EqualFold(k, "trevol-entorno") || strings.EqualFold(k, "trevol-escenario") {
			t.Errorf("cabecera que no debe ir: %s", k)
		}
	}
}

func TestAPIVersionEsConfigurableYViajaEnCadaPeticion(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveTest, []any{jsonResp(200, documento)}, []Option{WithAPIVersion("2026-10-21")})
	for i := 0; i < 2; i++ {
		b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	}
	for _, p := range b.peticiones() {
		if p.cabeceras.Get("Trevol-Version") != "2026-10-21" {
			t.Errorf("Trevol-Version = %q", p.cabeceras.Get("Trevol-Version"))
		}
	}
}

func TestElAcceptDeUnaDescargaEsElDelArchivoYLosBytesLleganIntactos(t *testing.T) {
	binario := []byte{37, 80, 68, 70, 0, 255, 1, 2}
	b := nuevoBanco(t, resp{estado: 200, crudo: binario, tipo: "application/pdf"})
	f, _, err := b.cliente.InvoicesAPI.DownloadInvoicePdf(context.Background(), idDoc).Execute()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got := make([]byte, 64)
	n, _ := f.Read(got)
	if string(got[:n]) != string(binario) {
		t.Errorf("bytes = %v", got[:n])
	}
	p := b.peticiones()[0]
	if p.cabeceras.Get("Accept") != "application/pdf, application/problem+json" || p.ruta != "/v1/invoices/"+idDoc+"/pdf" {
		t.Errorf("accept=%q ruta=%q", p.cabeceras.Get("Accept"), p.ruta)
	}
}

func TestLaLlaveLiveSeUsaTalCualYLoDiceElEntorno(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveLive, []any{jsonResp(200, documento)}, nil)
	b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	if b.peticiones()[0].cabeceras.Get("Authorization") != "Bearer "+llaveLive || b.cliente.Environment != EnvLive {
		t.Errorf("env=%v", b.cliente.Environment)
	}
}

func TestUnaCabeceraPreferDeLaPersonaPasa(t *testing.T) {
	b := nuevoBanco(t, jsonResp(200, documento))
	b.cliente.GetConfig().AddDefaultHeader("Prefer", "code=404")
	b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	if b.peticiones()[0].cabeceras.Get("Prefer") != "code=404" {
		t.Error("la cabecera Prefer no pasó")
	}
}

// ── Idempotency-Key ──
func TestGeneraUnaClaveYLaRepiteEnLosTresIntentos(t *testing.T) {
	b := nuevoBanco(t, problemaResp(503, "dian_unavailable", map[string]any{"retryable": true}, nil),
		problemaResp(503, "dian_unavailable", map[string]any{"retryable": true}, nil), jsonResp(201, documento))
	if _, err := crearFactura(t, b); err != nil {
		t.Fatal(err)
	}
	ps := b.peticiones()
	if len(ps) != 3 {
		t.Fatalf("%d peticiones", len(ps))
	}
	claves := map[string]bool{}
	cuerpos := map[string]bool{}
	for _, p := range ps {
		claves[p.cabeceras.Get("Idempotency-Key")] = true
		cuerpos[p.cuerpo] = true
	}
	if len(claves) != 1 || !claves["00000000-0000-4000-8000-000000000001"] {
		t.Errorf("claves = %v", claves)
	}
	if len(cuerpos) != 1 || !strings.Contains(ps[2].cuerpo, `"external_id":"VENTA-2026-000123"`) {
		t.Errorf("el cuerpo no viajó entero en cada intento: %v", cuerpos)
	}
	if len(b.esperas) != 2 {
		t.Errorf("esperas = %v", b.esperas)
	}
}

func TestRespetaLaClaveDeLaPersona(t *testing.T) {
	b := nuevoBanco(t, jsonResp(201, documento))
	crearFactura(t, b, "VENTA-2026-000123")
	if got := b.peticiones()[0].cabeceras.Get("Idempotency-Key"); got != "VENTA-2026-000123" {
		t.Errorf("clave = %q", got)
	}
}

func TestUnaClaveMalaSeRechazaLocalmenteSinRed(t *testing.T) {
	for _, mala := range []string{"con espacio", strings.Repeat("x", 256), "tilde-é"} {
		b := nuevoBanco(t, jsonResp(201, documento))
		_, err := crearFactura(t, b, mala)
		var te *Error
		if !errors.As(err, &te) || te.Kind != KindConfiguration {
			t.Errorf("%q: err = %v", mala, err)
		}
		if len(b.peticiones()) != 0 {
			t.Errorf("%q: salió a la red", mala)
		}
	}
}

func TestLaLlevaElErrorParaReintentarMasTardeConLaMisma(t *testing.T) {
	b := nuevoBanco(t, problemaResp(422, "dian_rejected", nil, nil))
	_, err := crearFactura(t, b)
	var te *Error
	if !errors.As(err, &te) || te.Kind != KindDianRejected || te.IdempotencyKey != "00000000-0000-4000-8000-000000000001" {
		t.Errorf("err = %#v", te)
	}
}

func TestCrearCorregirYLaRecargaLaLlevanElRestoNo(t *testing.T) {
	segunLaRuta := func(p peticion) resp {
		switch {
		case strings.HasSuffix(p.ruta, "/top_ups"):
			return jsonResp(201, map[string]any{"object": "top_up", "reference": "TRV-REC-1", "amount_cop": 50000, "status": "prepared"})
		case strings.HasSuffix(p.ruta, "/webhook_endpoints"):
			return jsonResp(201, map[string]any{"id": "we_" + strings.Repeat("a", 32), "object": "webhook_endpoint", "environment": "test"})
		case p.metodo == "POST" && strings.HasSuffix(p.ruta, "/invoices"):
			return jsonResp(201, documento)
		}
		return jsonResp(200, documento)
	}
	b := nuevoBanco(t, segunLaRuta)
	ctx := context.Background()
	c := b.cliente
	var entrada generado.InvoiceInput
	jsonDe(`{"external_id":"VENTA-2026-000123","customer":{"final_consumer":true},"items":[{"description":"Café","quantity":"1","unit_price":"1000"}]}`, &entrada)
	c.InvoicesAPI.CreateInvoice(ctx).InvoiceInput(entrada).Execute()
	c.InvoicesAPI.CorrectInvoice(ctx, idDoc).InvoiceInput(entrada).Execute()
	c.BalanceAPI.CreateBalanceTopUp(ctx).TopUpInput(*generado.NewTopUpInput(50000)).Execute()
	c.InvoicesAPI.RetrieveInvoice(ctx, idDoc).Execute()
	c.InvoicesAPI.SendInvoice(ctx, idDoc).Execute()
	c.WebhookEndpointsAPI.CreateWebhookEndpoint(ctx).WebhookEndpointInput(*generado.NewWebhookEndpointInput("https://tu-servidor.co/h", []string{"*"})).Execute()
	c.InvoicesAPI.PreviewInvoice(ctx).InvoiceInput(entrada).Execute()
	var vistas []string
	for _, p := range b.peticiones() {
		ruta := strings.Split(p.ruta, "?")[0]
		vistas = append(vistas, p.metodo+" "+ruta+" "+map[bool]string{true: "True", false: "False"}[p.cabeceras.Get("Idempotency-Key") != ""])
	}
	esperado := []string{"POST /v1/invoices True", "PUT /v1/invoices/" + idDoc + " True", "POST /v1/balance/top_ups True",
		"GET /v1/invoices/" + idDoc + " False", "POST /v1/invoices/" + idDoc + "/send False",
		"POST /v1/webhook_endpoints False", "POST /v1/invoices/preview False"}
	if strings.Join(vistas, "|") != strings.Join(esperado, "|") {
		t.Errorf("vistas = %v", vistas)
	}
}

func TestElTransporteNoReintentaPorSuCuentaNiConSegundoIntentoSinRebobinar(t *testing.T) {
	// C14: el SDK es el único que reintenta. Con max-retries 0, una caída es UN intento.
	b := nuevoBancoConLlave(t, llaveTest, []any{resp{cortar: true}}, []Option{WithMaxRetries(0)})
	_, _, err := b.cliente.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	var te *Error
	if !errors.As(err, &te) || te.Kind != KindConnection {
		t.Fatalf("err = %v", err)
	}
	if n := len(b.peticiones()); n != 1 {
		t.Errorf("peticiones = %d", n)
	}
}

// ── Llave y entorno ──
func TestEntornoDeLaLlaveYMalFormadas(t *testing.T) {
	c, err := NewClient(llaveTest)
	if err != nil || c.Environment != EnvTest {
		t.Fatalf("test: %v %v", c, err)
	}
	c, err = NewClient(llaveLive)
	if err != nil || c.Environment != EnvLive {
		t.Fatalf("live: %v %v", c, err)
	}
	malas := []string{"tvl_test_abc", "tvl_prod_" + strings.Repeat("a1", 32), "sk_live_" + strings.Repeat("a1", 32), "tvl_test_" + strings.Repeat("A1", 32),
		"tvl_test_" + strings.Repeat("a1", 33), "tvl_test_" + strings.Repeat("zz", 32)}
	for _, mala := range malas {
		_, err := NewClient(mala)
		var te *Error
		if !errors.As(err, &te) || te.Kind != KindConfiguration {
			t.Errorf("%q: %v", mala, err)
			continue
		}
		if strings.Contains(err.Error(), mala) {
			t.Errorf("el mensaje contiene la llave: %v", err)
		}
	}
}

func TestLeeTrevolAPIKeySiNoSePasa(t *testing.T) {
	t.Setenv("TREVOL_API_KEY", llaveLive)
	c, err := NewClient("")
	if err != nil || c.Environment != EnvLive {
		t.Fatalf("%v %v", c, err)
	}
	t.Setenv("TREVOL_API_KEY", "")
	if _, err := NewClient(""); err == nil {
		t.Error("sin llave y sin variable debía fallar")
	}
}

func TestEnvironmentQueNoCoincideEsErrorDeConfiguracionAntesDeLlamar(t *testing.T) {
	if _, err := NewClient(llaveTest, WithEnvironment(EnvLive)); err == nil || !strings.Contains(err.Error(), "live") {
		t.Errorf("err = %v", err)
	}
	if _, err := NewClient(llaveLive, WithEnvironment(EnvTest)); err == nil {
		t.Error("debía fallar")
	}
	if c, err := NewClient(llaveTest, WithEnvironment(EnvTest)); err != nil || c.Environment != EnvTest {
		t.Errorf("%v", err)
	}
}

// ── Cobertura de recursos ──
func TestCadaEtiquetaDeOpenAPITieneSuCampoEnElCliente(t *testing.T) {
	var etiquetas []string
	datos(t, "tags.json", &etiquetas)
	if strings.Join(etiquetas, ",") != strings.Join(generado.Etiquetas, ",") {
		t.Fatalf("tablas y especificación difieren: %v", etiquetas)
	}
	c, _ := NewClient(llaveTest)
	for _, e := range etiquetas {
		if !tieneCampo(c.APIClient, e+"API") {
			t.Errorf("falta %sAPI", e) // si la API suma un recurso, esto falla hasta agregarlo
		}
	}
}

func TestCadaOperationIDExisteComoMetodoEnSuRecurso(t *testing.T) {
	c, _ := NewClient(llaveTest)
	for _, o := range generado.Operaciones {
		if !tieneMetodo(c.APIClient, o.Recurso+"API", o.OperationID) {
			t.Errorf("falta %sAPI.%s", o.Recurso, o.OperationID)
		}
	}
}
