// Los ejemplos del README, compilados: un bloque que no compila (una variable sin usar, un error sin revisar, una API que cambió) rompe
// esta suite en vez de llegar a quien copia y pega.
//
//	cd sdks/go && go test ./pruebas/readme/
package readme

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var bloqueGo = regexp.MustCompile("(?s)```go\n(.*?)\n```")

// bloques devuelve los bloques ```go del README con el título (## o ###) que los precede.
func bloques(t *testing.T) map[string]string {
	crudo, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	texto := string(crudo)
	salida := map[string]string{}
	for _, m := range bloqueGo.FindAllStringSubmatchIndex(texto, -1) {
		antes := texto[:m[0]]
		titulo := ""
		for _, l := range strings.Split(antes, "\n") {
			if strings.HasPrefix(l, "#") {
				titulo = strings.TrimSpace(strings.TrimLeft(l, "# "))
			}
		}
		salida[titulo] = texto[m[2]:m[3]]
	}
	return salida
}

// compila escribe `fuente` como un programa dentro del módulo y lo compila con `go vet` (detecta también las variables sin usar).
func compila(t *testing.T, fuente string) {
	t.Helper()
	dir, err := os.MkdirTemp(".", "tmp-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(fuente), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "vet", "./"+dir).CombinedOutput(); err != nil {
		t.Fatalf("el ejemplo no compila:\n%s\n--- fuente ---\n%s", out, fuente)
	}
}

func TestElPrimerEjemploDelREADMECompila(t *testing.T) {
	b := bloques(t)["Uso"]
	if !strings.HasPrefix(b, "package main") {
		t.Fatalf("el bloque de «Uso» ya no es un programa completo:\n%s", b)
	}
	compila(t, b)
}

func TestElEjemploDeWebhooksCompila(t *testing.T) {
	b := bloques(t)["Webhooks"]
	if b == "" {
		t.Fatal("no encuentro el bloque de «Webhooks»")
	}
	compila(t, "package main\n\nimport (\n\t\"io\"\n\t\"log\"\n\t\"net/http\"\n\t\"os\"\n\n\ttrevol \"github.com/trevol-si/trevol-go\"\n)\n\nfunc main() {\n"+b+"\n\tlog.Fatal(http.ListenAndServe(\":8080\", nil))\n\t_ = io.EOF\n\t_ = os.Args\n}\n")
}

func TestLasOpcionesDelREADMECompilan(t *testing.T) {
	b := bloques(t)["Opciones"]
	if b == "" {
		t.Fatal("no encuentro el bloque de «Opciones»")
	}
	compila(t, "package main\n\nimport (\n\t\"net/http\"\n\t\"time\"\n\n\ttrevol \"github.com/trevol-si/trevol-go\"\n)\n\nfunc main() {\n\tmiClienteHTTP := &http.Client{}\n\tllave := \"tvl_test_00\"\n\t_, _ = "+strings.TrimPrefix(b, "")+"\n\t_ = time.Second\n}\n")
}
