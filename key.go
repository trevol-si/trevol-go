package trevol

import "regexp"

// Environment es el entorno que se deduce del prefijo de la llave: EnvTest (habilitación de la DIAN) o EnvLive (producción), el mismo
// vocabulario que `event.environment`.
type Environment string

const (
	EnvTest Environment = "test"
	EnvLive Environment = "live"
)

var reLlave = regexp.MustCompile(`^tvl_(test|live)_[0-9a-f]{64}$`)

// environmentOf valida la llave y devuelve su entorno. El mensaje NUNCA incluye la llave.
func environmentOf(llave string) (Environment, *Error) {
	m := reLlave.FindStringSubmatch(llave)
	if m == nil {
		return "", nuevoError(KindConfiguration, "La llave de la API no tiene el formato esperado (tvl_test_… o tvl_live_… con 64 caracteres hexadecimales). "+
			"Créala en https://platform.trevol.si/llaves y pásala completa o define TREVOL_API_KEY.")
	}
	return Environment(m[1]), nil
}
