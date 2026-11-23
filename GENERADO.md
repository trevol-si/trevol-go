# Código generado

**No edites `generado/`**: `sdks/generar.sh` la borra y la vuelve a escribir entera. La capa propia (cliente,
reintentos, errores, paginación, webhooks) vive fuera de esa carpeta.

| | |
|---|---|
| Generador | OpenAPI Generator 7.26.0 (`go`) |
| Especificación | `especificacion/openapi.sdk.json` (variante 3.0.3 de `docs/api/openapi.json`, versión de la API 2026-11-18) |
| sha256 de la especificación | `3ff67b7ae53d4f7c54fe60674daeb0cb615f6b08945fe468ae75408970f04951` |
| Configuración | `openapi-generator.yaml` |

Regenerar (desde el monorepo de Trevol): `sdks/generar.sh go`. Comprobar que lo del repositorio es lo que sale de la
especificación: `sdks/generar.sh --comprobar go`.
