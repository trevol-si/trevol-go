// Los fragmentos de la documentación, como programas (docs/api/43 §2.9): se COMPILAN y se CORREN contra Prism. Un fragmento que no
// compila, o que Prism rechaza porque el cuerpo no cumple la especificación, rompe esta suite.
//
//	sdks/herramientas/prism.sh -- bash -c 'cd sdks/go && go test ./pruebas/fragmentos/'
package fragmentos

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func operaciones(t *testing.T) []string {
	crudo, err := os.ReadFile(filepath.Join("..", "datos", "fragmentos.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Lenguajes  []string                     `json:"lenguajes"`
		Fragmentos map[string]map[string]string `json:"fragmentos"`
	}
	if err := json.Unmarshal(crudo, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Fragmentos) != 178 { // 0.4.0: + 19
		t.Fatalf("hay %d operaciones", len(d.Fragmentos))
	}
	var ops []string
	for op, porLenguaje := range d.Fragmentos {
		ops = append(ops, op)
		if !strings.Contains(porLenguaje["go"], "TREVOL_API_KEY") || !strings.Contains(porLenguaje["go"], "trevol.NewClient") {
			t.Errorf("%s: el fragmento de Go no usa trevol.NewClient(os.Getenv(\"TREVOL_API_KEY\"))", op)
		}
	}
	sort.Strings(ops)
	return ops
}

// Las operaciones de /v1/account piden una llave de administración: el SDK se niega a llamarlas con una tvl_ (KindConfiguration antes de
// la red), así que su programa corre con una tva_ de prueba. Prism no distingue una de otra (las dos son Bearer).
const llaveAdministracion = "tva_test_c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4"

func operacionesDeCuenta(t *testing.T) map[string]bool {
	crudo, err := os.ReadFile(filepath.Join("..", "datos", "operaciones.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ops []struct {
		OperationID string `json:"operationId"`
		Llave       string `json:"llave"`
	}
	if err := json.Unmarshal(crudo, &ops); err != nil {
		t.Fatal(err)
	}
	deCuenta := map[string]bool{}
	for _, o := range ops {
		if o.Llave == "administracion" {
			deCuenta[o.OperationID] = true
		}
	}
	if len(deCuenta) != 10 {
		t.Fatalf("hay %d operaciones de /v1/account", len(deCuenta))
	}
	return deCuenta
}

func TestLaEstructura(t *testing.T) {
	ops := operaciones(t)
	entradas, err := os.ReadDir("programas")
	if err != nil {
		t.Fatal(err)
	}
	var carpetas []string
	for _, e := range entradas {
		if e.IsDir() {
			carpetas = append(carpetas, e.Name())
		}
	}
	sort.Strings(carpetas)
	if strings.Join(carpetas, ",") != strings.Join(ops, ",") {
		t.Fatal("los programas no son exactamente las operaciones de la especificación")
	}
}

func TestLosFragmentosCompilanYCorrenContraPrism(t *testing.T) {
	ops := operaciones(t)
	deCuenta := operacionesDeCuenta(t)
	salida := t.TempDir()
	build := exec.Command("go", "build", "-o", salida, "./programas/...")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("los fragmentos no compilan:\n%s", out)
	}
	prism := os.Getenv("TREVOL_PRISM_URL")
	if prism == "" {
		t.Skip("necesita Prism: sdks/herramientas/prism.sh -- bash -c 'cd sdks/go && go test ./pruebas/fragmentos/'")
	}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			cmd := exec.Command(filepath.Join(salida, op))
			cmd.Env = os.Environ()
			if deCuenta[op] {
				cmd.Env = append(cmd.Env, "TREVOL_API_KEY="+llaveAdministracion) // la última gana
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
		})
	}
}
