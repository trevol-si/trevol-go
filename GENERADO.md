# Código generado

**No edites `generado/`**: `sdks/generar.sh` la borra y la vuelve a escribir entera. La capa propia (cliente,
reintentos, errores, paginación, webhooks) vive fuera de esa carpeta.

| | |
|---|---|
| Generador | OpenAPI Generator 7.26.0 (`go`) |
| Especificación | `especificacion/openapi.sdk.json` (variante 3.0.3 de `docs/api/openapi.json`, versión de la API 2026-11-18) |
| sha256 de la especificación | `718dd7fdbb3f31d9c907376150fb663dbe33fcb27031a195c28db3917b270eb7` |
| Configuración | `openapi-generator.yaml` |

Regenerar (desde el monorepo de Trevol): `sdks/generar.sh go`. Comprobar que lo del repositorio es lo que sale de la
especificación: `sdks/generar.sh --comprobar go`.
