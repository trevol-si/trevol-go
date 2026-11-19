package trevol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trevol-si/trevol-go/generado"
)

const (
	llaveCuentaTest = "tva_test_c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4"
	llaveCuentaLive = "tva_live_d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5"
	empresaA        = "com_3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b"
	empresaB        = "com_9e8d7c6b9e8d7c6b9e8d7c6b9e8d7c6b"
)

var empresaCuenta = map[string]any{"id": empresaA, "object": "company", "nit": "900373427", "check_digit": "1",
	"name": "DISTRIBUIDORA LOS ANDES SAS", "trade_name": nil, "status": "pending",
	"onboarding": map[string]any{"step": "not_started", "stop_reason": nil}, "emission_environment": nil,
	"created_at": "2026-11-18T15:04:05Z", "enabled_at": nil,
	"onboarding_url": "https://app.trevol.si/empresas/3f0c1a2b-3f0c-1a2b-3f0c-1a2b3f0c1a2b",
	"api_keys":       map[string]any{"active_test": 0, "active_live": 0}}

func comoConfiguracion(t *testing.T, err error) *Error {
	t.Helper()
	var te *Error
	if !errors.As(err, &te) || te.Kind != KindConfiguration {
		t.Fatalf("se esperaba KindConfiguration: %T %v", err, err)
	}
	return te
}

func sinCabecerasDeEmpresa(t *testing.T, p peticion) {
	t.Helper()
	if v := p.cabeceras.Values("Trevol-Company"); len(v) > 0 {
		t.Errorf("%s %s: Trevol-Company = %v", p.metodo, p.ruta, v)
	}
	if v := p.cabeceras.Values("Trevol-Environment"); len(v) > 0 {
		t.Errorf("%s %s: Trevol-Environment = %v", p.metodo, p.ruta, v)
	}
}

// ── La llave ──

func TestAceptaLasLlavesDeAdministracionYDeduceEntornoYClase(t *testing.T) {
	casos := []struct {
		llave   string
		entorno Environment
		clase   KeyType
	}{{llaveTest, EnvTest, KeyTypeAPI}, {llaveLive, EnvLive, KeyTypeAPI}, {llaveCuentaTest, EnvTest, KeyTypeAccount}, {llaveCuentaLive, EnvLive, KeyTypeAccount}}
	for _, c := range casos {
		cl, err := NewClient(c.llave)
		if err != nil {
			t.Fatalf("%s…: %v", c.llave[:9], err)
		}
		if cl.Environment != c.entorno || cl.KeyType != c.clase || cl.Company() != "" {
			t.Errorf("%s…: env=%s clase=%s company=%q", c.llave[:9], cl.Environment, cl.KeyType, cl.Company())
		}
	}
}

func TestLasLlavesDeAdministracionMalFormadasSonErrorDeConfiguracionSinLaLlave(t *testing.T) {
	malas := []string{"tva_test_abc", "tva_prod_" + strings.Repeat("c4", 32), "tva_test_" + strings.Repeat("C4", 32),
		"tva_live_" + strings.Repeat("c4", 33), "tvx_test_" + strings.Repeat("c4", 32), "TVA_TEST_" + strings.Repeat("c4", 32),
		"tva_" + strings.Repeat("c4", 32)}
	for _, mala := range malas {
		_, err := NewClient(mala)
		te := comoConfiguracion(t, err)
		if strings.Contains(err.Error(), mala) || strings.Contains(fmt.Sprintf("%+v %#v", te, te), mala) {
			t.Errorf("el error contiene la llave: %v", err)
		}
		for _, prefijo := range []string{"tvl_test_", "tvl_live_", "tva_test_", "tva_live_"} {
			if !strings.Contains(err.Error(), prefijo) {
				t.Errorf("el mensaje no menciona %s: %v", prefijo, err)
			}
		}
		if !strings.Contains(err.Error(), "https://platform.trevol.si/llaves y la de administración en https://platform.trevol.si/administracion") {
			t.Errorf("el mensaje no dice dónde se crea cada llave: %v", err)
		}
	}
}

func TestWithEnvironmentConUnaLlaveDeAdministracion(t *testing.T) {
	_, err := NewClient(llaveCuentaTest, WithEnvironment(EnvLive))
	comoConfiguracion(t, err)
	if !strings.Contains(err.Error(), "tva_test_") || strings.Contains(err.Error(), llaveCuentaTest) {
		t.Errorf("mensaje: %v", err)
	}
	if c, err := NewClient(llaveCuentaLive, WithEnvironment(EnvLive)); err != nil || c.Environment != EnvLive {
		t.Errorf("%v", err)
	}
}

func TestLeeUnaLlaveDeAdministracionDeTrevolAPIKey(t *testing.T) {
	t.Setenv("TREVOL_API_KEY", llaveCuentaLive)
	c, err := NewClient("", WithCompany(empresaA))
	if err != nil || c.KeyType != KeyTypeAccount || c.Environment != EnvLive || c.Company() != empresaA {
		t.Fatalf("%v %v", c, err)
	}
}

// ── WithCompany y ForCompany ──

func TestWithCompanyValidaElIdLocalmente(t *testing.T) {
	c, err := NewClient(llaveCuentaTest, WithCompany("  "+empresaA+"\n"))
	if err != nil || c.Company() != empresaA {
		t.Fatalf("con espacios alrededor: %v %v", c, err)
	}
	for _, malo := range []string{"", "com_3F0C1A2B3F0C1A2B3F0C1A2B3F0C1A2B", "com_123", "inv_" + hex32, "3f0c1a2b3f0c1a2b3f0c1a2b3f0c1a2b", llaveTest} {
		_, err := NewClient(llaveCuentaTest, WithCompany(malo))
		comoConfiguracion(t, err)
		if malo != "" && strings.Contains(err.Error(), malo) {
			t.Errorf("el mensaje repite el valor: %v", err)
		}
		if !strings.Contains(err.Error(), "com_") {
			t.Errorf("el mensaje no explica el formato: %v", err)
		}
	}
}

func TestCompanyConUnaLlaveTvlEsErrorDeConfiguracion(t *testing.T) {
	_, err := NewClient(llaveTest, WithCompany(empresaA))
	comoConfiguracion(t, err)
	if !strings.Contains(err.Error(), "company solo aplica a una llave de administración") {
		t.Errorf("mensaje: %v", err)
	}
	c, _ := NewClient(llaveLive)
	derivado, err := c.ForCompany(empresaA)
	if derivado != nil {
		t.Error("ForCompany con tvl_ devolvió un cliente")
	}
	comoConfiguracion(t, err)
}

func TestForCompanyComparteElTransporteYNoTocaAlOriginal(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveCuentaTest, []any{jsonResp(200, documento)}, []Option{WithCompany(empresaA)})
	b.cliente.GetConfig().AddDefaultHeader("X-Mio", "1")
	otro, err := b.cliente.ForCompany(empresaB)
	if err != nil {
		t.Fatal(err)
	}
	if otro.Company() != empresaB || b.cliente.Company() != empresaA || otro.KeyType != KeyTypeAccount || otro.Environment != EnvTest {
		t.Fatalf("company: %q / %q", otro.Company(), b.cliente.Company())
	}
	if otro.t.base != b.cliente.t.base {
		t.Error("el derivado no comparte el transporte subyacente (las conexiones)")
	}
	otro.GetConfig().AddDefaultHeader("X-Solo-Derivado", "1")
	if _, ok := b.cliente.GetConfig().DefaultHeader["X-Solo-Derivado"]; ok {
		t.Error("las cabeceras del derivado se filtraron al original")
	}
	if _, err := otro.ForCompany("com_malo"); err == nil {
		t.Error("ForCompany con un id mal formado debía fallar")
	}
	ctx := context.Background()
	b.cliente.InvoicesAPI.RetrieveInvoice(ctx, idDoc).Execute()
	otro.InvoicesAPI.RetrieveInvoice(ctx, idDoc).Execute()
	b.cliente.InvoicesAPI.RetrieveInvoice(ctx, idDoc).Execute()
	ps := b.peticiones()
	if len(ps) != 3 {
		t.Fatalf("%d peticiones", len(ps))
	}
	for i, quiere := range []string{empresaA, empresaB, empresaA} {
		if got := ps[i].cabeceras.Get("Trevol-Company"); got != quiere {
			t.Errorf("petición %d: Trevol-Company = %q, quiero %q", i, got, quiere)
		}
		if ps[i].cabeceras.Get("X-Mio") != "1" || ps[i].cabeceras.Get("Authorization") != "Bearer "+llaveCuentaTest {
			t.Errorf("petición %d: cabeceras %v", i, ps[i].cabeceras)
		}
	}
	if ps[1].cabeceras.Get("X-Solo-Derivado") != "1" || ps[0].cabeceras.Get("X-Solo-Derivado") != "" {
		t.Error("X-Solo-Derivado")
	}
}

func TestForCompanyEnParaleloCadaUnoConSuEmpresa(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveCuentaLive, []any{jsonResp(200, documento)}, nil)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("com_%032x", i)
			c, err := b.cliente.ForCompany(id)
			if err != nil {
				t.Error(err)
				return
			}
			c.InvoicesAPI.RetrieveInvoice(context.Background(), fmt.Sprintf("inv_%032x", i)).Execute()
		}(i)
	}
	wg.Wait()
	ps := b.peticiones()
	if len(ps) != 16 {
		t.Fatalf("%d peticiones", len(ps))
	}
	for _, p := range ps {
		quiere := "com_" + strings.TrimPrefix(p.ruta, "/v1/invoices/inv_")
		if p.cabeceras.Get("Trevol-Company") != quiere || p.cabeceras.Get("Trevol-Environment") != "live" {
			t.Errorf("se mezclaron las empresas: %q para %q", p.cabeceras.Get("Trevol-Company"), p.ruta)
		}
	}
}

// ── Las cabeceras en cada ruta ──

func TestUnaRutaDeEmpresaConLlaveDeAdministracionLlevaLasDosCabeceras(t *testing.T) {
	for _, c := range []struct {
		llave   string
		entorno string
	}{{llaveCuentaTest, "test"}, {llaveCuentaLive, "live"}} {
		b := nuevoBancoConLlave(t, c.llave, []any{jsonResp(201, documento)}, []Option{WithCompany(empresaA)})
		if _, err := crearFactura(t, b); err != nil {
			t.Fatal(err)
		}
		h := b.peticiones()[0].cabeceras
		if h.Get("Authorization") != "Bearer "+c.llave || h.Get("Trevol-Company") != empresaA || h.Get("Trevol-Environment") != c.entorno {
			t.Errorf("%s: %v", c.entorno, h)
		}
		if h.Get("Idempotency-Key") == "" || h.Get("Trevol-Version") != APIVersion {
			t.Errorf("faltan las cabeceras de siempre: %v", h)
		}
	}
}

func TestSinEmpresaUnaRutaDeEmpresaFallaAntesDeLaRed(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveCuentaTest, []any{jsonResp(201, documento)}, nil)
	_, err := crearFactura(t, b)
	te := comoConfiguracion(t, err)
	if len(b.peticiones()) != 0 {
		t.Error("salió a la red")
	}
	for _, debe := range []string{"Trevol-Company", "createInvoice", "trevol.WithCompany", "ForCompany"} {
		if !strings.Contains(te.Error(), debe) {
			t.Errorf("el mensaje no dice %q: %v", debe, te)
		}
	}
	if strings.Contains(err.Error(), llaveCuentaTest) {
		t.Error("el error trae la llave")
	}
	// Las rutas de la cuenta sí funcionan sin empresa.
	b2 := nuevoBancoConLlave(t, llaveCuentaTest, []any{jsonResp(200, map[string]any{"object": "account_key_info"})}, nil)
	if _, _, err := b2.cliente.AccountAPI.RetrieveAccount(context.Background()).Execute(); err != nil {
		t.Fatal(err)
	}
	sinCabecerasDeEmpresa(t, b2.peticiones()[0])
}

// Una llave tvl_ es de una sola empresa: las operaciones de AccountAPI (solo admiten una tva_) fallan con KindConfiguration antes de la
// red, sin la llave en el mensaje, como en los demás SDK. Un cuerpo ya armado (CreateAccountCompany) no sale.
func TestAccountAPIConUnaLlaveTvlFallaAntesDeLaRed(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveTest, []any{jsonResp(200, map[string]any{"object": "account_key_info"})}, nil)
	_, _, err := b.cliente.AccountAPI.RetrieveAccount(context.Background()).Execute()
	te := comoConfiguracion(t, err)
	for _, debe := range []string{"AccountAPI", "tva_test_", "tva_live_", "tvl_", "retrieveAccount"} {
		if !strings.Contains(te.Error(), debe) {
			t.Errorf("el mensaje no dice %q: %v", debe, te)
		}
	}
	if strings.Contains(err.Error(), llaveTest) {
		t.Error("el error trae la llave")
	}
	entrada := generado.NewAccountCompanyInput("900373427")
	_, _, err = b.cliente.AccountAPI.CreateAccountCompany(context.Background()).AccountCompanyInput(*entrada).Execute()
	comoConfiguracion(t, err)
	if len(b.peticiones()) != 0 {
		t.Error("salió a la red")
	}
	// Las rutas de la empresa siguen funcionando con esa misma llave.
	b2 := nuevoBancoConLlave(t, llaveTest, []any{jsonResp(201, documento)}, nil)
	if _, err := crearFactura(t, b2); err != nil {
		t.Fatal(err)
	}
	sinCabecerasDeEmpresa(t, b2.peticiones()[0])
}

// Cada operación de la tabla, con la llave que admite: tvl_ nunca manda las cabeceras (y en /account no sale: KindConfiguration);
// tva_ solo en las de empresa; /account y /health nunca. La petición va por el transporte del cliente (el mismo camino que los métodos
// generados).
func TestCadaOperacionDeLaTablaLlevaLasCabecerasQueLeTocan(t *testing.T) {
	llaves := map[string]int{}
	for _, llave := range []string{llaveTest, llaveCuentaTest} {
		opts := []Option{WithMaxRetries(0)}
		if llave == llaveCuentaTest {
			opts = append(opts, WithCompany(empresaA))
		}
		b := nuevoBancoConLlave(t, llave, []any{jsonResp(200, map[string]any{})}, opts)
		b.cliente.GetConfig().AddDefaultHeader("Trevol-Company", empresaB) // el SDK manda: se reemplaza o se quita
		b.cliente.GetConfig().AddDefaultHeader("Trevol-Environment", "live")
		for _, op := range generado.Operaciones {
			ruta := strings.NewReplacer("{id}", "x_1", "{key_id}", "key_1", "{cufe}", "abc", "{run_id}", "x_2").Replace(op.Ruta)
			req, _ := http.NewRequest(op.Metodo, b.srv.URL+"/v1"+ruta, nil)
			for k, v := range b.cliente.GetConfig().DefaultHeader {
				req.Header.Set(k, v)
			}
			antes := len(b.peticiones())
			r, err := b.cliente.GetConfig().HTTPClient.Do(req)
			if llave == llaveTest {
				llaves[op.Llave]++
			}
			if llave == llaveTest && op.Llave == "administracion" {
				// Con una tvl_, una operación de /v1/account no sale: la API la rechazaría con 401.
				var te *Error
				if !errors.As(err, &te) || te.Kind != KindConfiguration || !strings.Contains(te.Error(), "tva_test_") ||
					strings.Contains(err.Error(), llaveTest) {
					t.Errorf("%s: %v", op.OperationID, err)
				}
				if len(b.peticiones()) != antes {
					t.Errorf("%s: la petición salió a la red", op.OperationID)
				}
				continue
			}
			if err != nil {
				t.Fatalf("%s: %v", op.OperationID, err)
			}
			r.Body.Close()
			p := b.peticiones()[len(b.peticiones())-1]
			if llave == llaveCuentaTest && op.Llave == "api_o_administracion" {
				if p.cabeceras.Get("Trevol-Company") != empresaA || p.cabeceras.Get("Trevol-Environment") != "test" ||
					len(p.cabeceras.Values("Trevol-Company")) != 1 {
					t.Errorf("%s: %v", op.OperationID, p.cabeceras)
				}
				continue
			}
			sinCabecerasDeEmpresa(t, p)
		}
	}
	if llaves["administracion"] != 10 || llaves["ninguna"] != 1 || llaves["api_o_administracion"] != len(generado.Operaciones)-11 {
		t.Errorf("la tabla cambió: %v", llaves)
	}
}

func TestUnaRutaQueNoEstaEnLaTablaSeClasificaPorSuForma(t *testing.T) {
	casos := map[string]string{"/account": "administracion", "/account/companies/x/algo_nuevo": "administracion", "/health": "ninguna",
		"/accounting": "api_o_administracion", "/recurso_nuevo": "api_o_administracion"}
	for ruta, quiere := range casos {
		if got := llaveDeRuta(ruta); got != quiere {
			t.Errorf("%s: %s", ruta, got)
		}
	}
	b := nuevoBancoConLlave(t, llaveCuentaTest, []any{jsonResp(200, map[string]any{})}, nil)
	req, _ := http.NewRequest("GET", b.srv.URL+"/v1/recurso_nuevo", nil)
	_, err := b.cliente.GetConfig().HTTPClient.Do(req)
	comoConfiguracion(t, err)
	if len(b.peticiones()) != 0 {
		t.Error("salió a la red")
	}
}

// ── El recurso de cuenta ──

func TestAccountAPIConPaginacionYSinCabecerasDeEmpresa(t *testing.T) {
	otra := map[string]any{}
	for k, v := range empresaCuenta {
		otra[k] = v
	}
	otra["id"] = empresaB
	paginas := func(p peticion) resp {
		if strings.Contains(p.ruta, "starting_after="+empresaA) {
			return jsonResp(200, map[string]any{"object": "list", "data": []any{otra}, "has_more": false, "next_cursor": nil})
		}
		return jsonResp(200, map[string]any{"object": "list", "data": []any{empresaCuenta}, "has_more": true, "next_cursor": empresaA})
	}
	b := nuevoBancoConLlave(t, llaveCuentaLive, []any{paginas}, []Option{WithCompany(empresaA)})
	ctx := context.Background()
	var ids []string
	for e, err := range Paginate(ctx, func(cursor *string) (*generado.AccountCompanyList, error) {
		r := b.cliente.AccountAPI.ListAccountCompanies(ctx).Limit(100).Status("enabled")
		if cursor != nil {
			r = r.StartingAfter(*cursor)
		}
		l, _, err := r.Execute()
		return l, err
	}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, e.GetId())
	}
	if strings.Join(ids, ",") != empresaA+","+empresaB {
		t.Fatalf("ids = %v", ids)
	}
	ps := b.peticiones()
	if len(ps) != 2 || !strings.HasPrefix(ps[0].ruta, "/v1/account/companies?") || !strings.Contains(ps[1].ruta, "starting_after="+empresaA) {
		t.Fatalf("rutas: %v", ps)
	}
	for _, p := range ps {
		sinCabecerasDeEmpresa(t, p)
		if p.cabeceras.Get("Authorization") != "Bearer "+llaveCuentaLive {
			t.Error("Authorization")
		}
	}
}

func TestLasListasDeLaCuentaSirvenConPaginate(t *testing.T) {
	// Compila con cada sobre de lista nuevo: si el generador cambiara su forma, esto deja de compilar.
	ctx := context.Background()
	c, _ := NewClient(llaveCuentaTest)
	_ = Paginate(ctx, func(*string) (*generado.AccountApiKeyList, error) { return nil, nil })
	_ = Paginate(ctx, func(*string) (*generado.AccountDianDocumentList, error) { return nil, nil })
	_ = Paginate(ctx, func(*string) (*generado.AccountRadianEventList, error) { return nil, nil })
	_ = Paginate(ctx, func(*string) (*generado.AccountRequestLogList, error) { return nil, nil })
	_ = Paginate(ctx, func(*string) (*generado.AccountCompanyList, error) { return nil, nil })
	if c.AccountAPI == nil {
		t.Fatal("falta AccountAPI")
	}
	for _, m := range []string{"RetrieveAccount", "ListAccountCompanies", "CreateAccountCompany", "RetrieveAccountCompany",
		"ListAccountCompanyApiKeys", "CreateAccountCompanyApiKey", "RevokeAccountCompanyApiKey", "ListAccountDianDocuments",
		"ListAccountRadianEvents", "ListAccountRequests"} {
		if !tieneMetodo(c.APIClient, "AccountAPI", m) {
			t.Errorf("falta AccountAPI.%s", m)
		}
	}
}

func TestCrearEmpresaLlevaIdempotencyKeyYCompanyExistsEsConflicto(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveCuentaLive, []any{problemaResp(409, "company_exists", map[string]any{"company": empresaCuenta}, nil)}, nil)
	entrada := generado.NewAccountCompanyInput("900373427")
	_, _, err := b.cliente.AccountAPI.CreateAccountCompany(context.Background()).AccountCompanyInput(*entrada).Execute()
	e := comoError(t, err)
	if e.Kind != KindConflict || e.Code != "company_exists" || e.Problem["company"].(map[string]any)["id"] != empresaA {
		t.Errorf("%+v", e)
	}
	p := b.peticiones()[0]
	if p.metodo != "POST" || p.ruta != "/v1/account/companies" || p.cabeceras.Get("Idempotency-Key") == "" || e.IdempotencyKey == "" ||
		!strings.Contains(p.cuerpo, `"nit":"900373427"`) {
		t.Errorf("%+v", p)
	}
	sinCabecerasDeEmpresa(t, p)
	if len(b.peticiones()) != 1 {
		t.Error("company_exists no se reintenta")
	}
}

func TestCompanyLimitReachedEsRateLimitYNoEsperaUnaHora(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveCuentaLive, []any{problemaResp(429, "company_limit_reached",
		map[string]any{"retryable": true, "retry_after": 3600, "limit": "hour"}, map[string]string{"Retry-After": "3600"})}, nil)
	_, _, err := b.cliente.AccountAPI.CreateAccountCompany(context.Background()).AccountCompanyInput(*generado.NewAccountCompanyInput("900373427")).Execute()
	e := comoError(t, err)
	if e.Kind != KindRateLimit || e.RetryAfter != time.Hour || e.Problem["limit"] != "hour" {
		t.Errorf("%+v", e)
	}
	if len(b.peticiones()) != 1 || len(b.esperas) != 0 {
		t.Errorf("peticiones=%d esperas=%v", len(b.peticiones()), b.esperas)
	}
}

// Al repetirla con la misma Idempotency-Key la API devuelve la llave con token null: un reintento a ciegas perdería el token.
func TestCrearUnaLlaveDeEmpresaNoSeReintentaAnteUn5xxNiUnCorteYSiAnteUn429(t *testing.T) {
	for _, o := range generado.Operaciones {
		if o.OperationID == "createAccountCompanyApiKey" && (!o.Idempotente || o.Reintento != "solo_429") {
			t.Errorf("la tabla: %+v", o)
		}
	}
	crear := func(b *banco) (*generado.AccountApiKey, error) {
		k, _, err := b.cliente.AccountAPI.CreateAccountCompanyApiKey(context.Background(), empresaA).
			AccountApiKeyInput(*generado.NewAccountApiKeyInput("ERP de la bodega", "write")).Execute()
		return k, err
	}
	a := nuevoBancoConLlave(t, llaveCuentaTest, []any{problemaResp(503, "dian_unavailable", map[string]any{"retryable": true}, nil)}, nil)
	_, err := crear(a)
	if e := comoError(t, err); e.Kind != KindDianUnavailable || e.IdempotencyKey == "" || len(a.peticiones()) != 1 || len(a.esperas) != 0 {
		t.Errorf("503: kind=%s clave=%q intentos=%d esperas=%v", e.Kind, e.IdempotencyKey, len(a.peticiones()), a.esperas)
	}
	b := nuevoBancoConLlave(t, llaveCuentaTest, []any{resp{cortar: true}}, nil)
	_, err = crear(b)
	if comoError(t, err).Kind != KindConnection || len(b.peticiones()) != 1 {
		t.Errorf("corte: intentos=%d", len(b.peticiones()))
	}
	c := nuevoBancoConLlave(t, llaveCuentaTest, []any{
		problemaResp(429, "rate_limited", map[string]any{"retryable": true, "limit": "api_keys"}, map[string]string{"Retry-After": "2"}),
		jsonResp(201, map[string]any{"id": "key_" + hex32, "object": "api_key", "token": "tvl_test_" + strings.Repeat("e7", 32)}),
	}, nil)
	k, err := crear(c)
	if err != nil || !strings.HasPrefix(k.GetToken(), "tvl_test_") {
		t.Fatalf("429: %v", err)
	}
	if len(c.peticiones()) != 2 || len(c.esperas) != 1 || c.esperas[0] != 2*time.Second {
		t.Errorf("429: intentos=%d esperas=%v", len(c.peticiones()), c.esperas)
	}
	if c.peticiones()[0].cabeceras.Get("Idempotency-Key") != c.peticiones()[1].cabeceras.Get("Idempotency-Key") {
		t.Error("el reintento cambió la Idempotency-Key")
	}
}

func TestRevocarUnaLlaveEsDeleteSinIdempotencyKey(t *testing.T) {
	b := nuevoBancoConLlave(t, llaveCuentaTest, []any{jsonResp(200, map[string]any{"id": "key_" + hex32, "object": "api_key", "status": "revoked"})}, nil)
	r, _, err := b.cliente.AccountAPI.RevokeAccountCompanyApiKey(context.Background(), empresaA, "key_"+hex32).Execute()
	if err != nil || r.GetStatus() != "revoked" {
		t.Fatalf("%v %v", r, err)
	}
	p := b.peticiones()[0]
	if p.metodo != "DELETE" || p.ruta != "/v1/account/companies/"+empresaA+"/api_keys/key_"+hex32 || p.cabeceras.Get("Idempotency-Key") != "" {
		t.Errorf("%+v", p)
	}
	sinCabecerasDeEmpresa(t, p)
}

// ── La llave nunca se ve ──

func TestLaLlaveNoSaleAlImprimirElClienteNiSuTransporte(t *testing.T) {
	for _, llave := range []string{llaveTest, llaveCuentaLive} {
		opts := []Option{}
		if llave == llaveCuentaLive {
			opts = append(opts, WithCompany(empresaA))
		}
		c, err := NewClient(llave, opts...)
		if err != nil {
			t.Fatal(err)
		}
		vistas := []any{c, *c, c.GetConfig(), *c.GetConfig(), c.GetConfig().HTTPClient, *c.GetConfig().HTTPClient, c.GetConfig().HTTPClient.Transport, c.t, c.t.o}
		if llave == llaveCuentaLive {
			d, _ := c.ForCompany(empresaB)
			vistas = append(vistas, d, *d)
		}
		for i, v := range vistas {
			for _, verbo := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
				if texto := fmt.Sprintf(verbo, v); strings.Contains(texto, llave) || strings.Contains(texto, llave[9:]) ||
					strings.Contains(texto, fmt.Sprintf("%x", llave)) {
					t.Errorf("vista %d con %s muestra la llave: %s", i, verbo, texto)
				}
			}
		}
		if !strings.Contains(c.String(), llave[:9]+"…") {
			t.Errorf("String no dice la clase de llave: %s", c)
		}
	}
}

func TestLaLlaveNoSaleEnElRegistroDeDepuracionNiEnLosErrores(t *testing.T) {
	var registro bytes.Buffer
	anterior, banderas := log.Writer(), log.Flags()
	log.SetOutput(&registro)
	defer func() { log.SetOutput(anterior); log.SetFlags(banderas) }()

	b := nuevoBancoConLlave(t, llaveCuentaTest, []any{problemaResp(401, "invalid_api_key", nil, nil)}, []Option{WithCompany(empresaA)})
	b.cliente.GetConfig().Debug = true
	_, err := leerFactura(b)
	e := comoError(t, err)
	if e.Kind != KindAuthentication {
		t.Fatalf("%v", e)
	}
	if registro.Len() == 0 {
		t.Fatal("el modo Debug del generador no escribió nada: la prueba no prueba nada")
	}
	textos := []string{registro.String(), err.Error(), fmt.Sprintf("%+v %#v", e, e), fmt.Sprint(e.Header), fmt.Sprint(e.Problem)}
	// Un fallo de red tampoco la lleva.
	c, _ := NewClient(llaveCuentaTest, WithCompany(empresaA), WithBaseURL("http://127.0.0.1:1/v1"), WithMaxRetries(0))
	_, _, errRed := c.InvoicesAPI.RetrieveInvoice(context.Background(), idDoc).Execute()
	textos = append(textos, errRed.Error(), fmt.Sprintf("%+v", errRed))
	for _, s := range textos {
		if strings.Contains(s, llaveCuentaTest) || strings.Contains(s, llaveCuentaTest[9:]) {
			t.Errorf("la llave aparece: %s", s)
		}
	}
}

func TestUnClienteVacioNoRevientaAlImprimirse(t *testing.T) {
	var c *Client
	if c.Company() != "" || fmt.Sprint(c) != "trevol.Client{}" || fmt.Sprint(&Client{}) != "trevol.Client{}" {
		t.Error("cliente vacío")
	}
	if _, err := (&Client{}).ForCompany(empresaA); err == nil {
		t.Error("ForCompany sobre un cliente vacío")
	}
}
