package trevol

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/trevol/trevol-go/generado"
)

func jsonDe(texto string, destino any) error {
	return json.NewDecoder(strings.NewReader(texto)).Decode(destino)
}

func tieneCampo(c *generado.APIClient, nombre string) bool {
	return reflect.ValueOf(c).Elem().FieldByName(nombre).IsValid()
}

func tieneMetodo(c *generado.APIClient, campo, metodo string) bool {
	f := reflect.ValueOf(c).Elem().FieldByName(campo)
	if !f.IsValid() {
		return false
	}
	return f.MethodByName(strings.ToUpper(metodo[:1]) + metodo[1:]).IsValid()
}
