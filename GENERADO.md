# Código generado

**No edites `generado/`**: `sdks/generar.sh` la borra y la vuelve a escribir entera. La capa propia (cliente,
reintentos, errores, paginación, webhooks) vive fuera de esa carpeta.

| | |
|---|---|
| Generador | OpenAPI Generator 7.26.0 (`go`) |
| Especificación | `especificacion/openapi.sdk.json` (variante 3.0.3 de `docs/api/openapi.json`, versión de la API 2026-11-18) |
| sha256 de la especificación | `89e9dc9a871d7f301a7f7f10a9d6031bc07b5dae1bf400d640949162ec2b4ecb` |
| Configuración | `openapi-generator.yaml` |

Regenerar (desde el monorepo de Trevol): `sdks/generar.sh go`. Comprobar que lo del repositorio es lo que sale de la
especificación: `sdks/generar.sh --comprobar go`.
