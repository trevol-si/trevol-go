package trevol

import (
	"regexp"
	"strings"
)

// Environment es el entorno que se deduce del prefijo de la llave: EnvTest (habilitación de la DIAN) o EnvLive (producción), el mismo
// vocabulario que `event.environment`.
type Environment string

const (
	EnvTest Environment = "test"
	EnvLive Environment = "live"
)

// KeyType es la clase de llave, deducida de su prefijo: KeyTypeAPI (tvl_…, la llave de UNA empresa) o KeyTypeAccount (tva_…, la llave
// de administración de una cuenta, que sirve para varias empresas).
type KeyType string

const (
	KeyTypeAPI     KeyType = "api"
	KeyTypeAccount KeyType = "account"
)

var (
	reLlave   = regexp.MustCompile(`^(tvl|tva)_(test|live)_[0-9a-f]{64}$`)
	reEmpresa = regexp.MustCompile(`^com_[0-9a-f]{32}$`)
)

// analizarLlave valida la llave y devuelve su entorno y su clase. El mensaje NUNCA incluye la llave.
func analizarLlave(llave string) (Environment, KeyType, *Error) {
	m := reLlave.FindStringSubmatch(llave)
	if m == nil {
		return "", "", nuevoError(KindConfiguration, "La llave de la API no tiene el formato esperado (tvl_test_…, tvl_live_…, tva_test_… o tva_live_… con 64 "+
			"caracteres hexadecimales). La de una empresa se crea en https://platform.trevol.si/llaves y la de administración en "+
			"https://platform.trevol.si/administracion; pásala completa o define TREVOL_API_KEY.")
	}
	clase := KeyTypeAPI
	if m[1] == "tva" {
		clase = KeyTypeAccount
	}
	return Environment(m[2]), clase, nil
}

// validarEmpresa revisa el id de empresa que acompaña a una llave de administración. Ni el mensaje ni el error repiten el valor (por si
// alguien pegó ahí una llave).
func validarEmpresa(clase KeyType, id string) (string, *Error) {
	if clase != KeyTypeAccount {
		return "", nuevoError(KindConfiguration, "company solo aplica a una llave de administración (tva_…): una llave tvl_ ya es de una empresa.")
	}
	id = strings.TrimSpace(id)
	if !reEmpresa.MatchString(id) {
		return "", nuevoError(KindConfiguration, "El id de la empresa no tiene el formato esperado (com_ y 32 caracteres hexadecimales en minúscula, "+
			"como sale en GET /v1/account/companies).")
	}
	return id, nil
}

// llaveVisible es lo único de la llave que el SDK muestra al imprimir un cliente: el prefijo de clase y entorno.
func llaveVisible(llave string) string {
	if m := reLlave.FindStringSubmatch(llave); m != nil {
		return m[1] + "_" + m[2] + "_…"
	}
	return "…"
}
