package trevol

import (
	"crypto/rand"
	"encoding/hex"
)

// nuevoUUID devuelve un UUID v4 (RFC 4122) con crypto/rand: la Idempotency-Key automática.
func nuevoUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("trevol: no hay fuente de azar del sistema: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
