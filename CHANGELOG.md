# Registro de cambios

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el versionado, [SemVer](https://semver.org/lang/es/): antes
de la 1.0, un cambio incompatible sube el número menor.

## [0.3.0] — 2026-11-19

Regenerado con la llave de administración pública (la misma versión `2026-11-18` de la API; 159 operaciones y 70 tipos de evento).

- Recurso nuevo `AccountAPI`: las diez operaciones de `/v1/account` para quien lleva varias empresas en una cuenta (la llave, sus empresas, reservar una empresa, las llaves de cada empresa y lo emitido, lo recibido y los eventos RADIAN de todas). Piden una llave de administración (`tva_test_…` o `tva_live_…`).
- Errores tipados para los códigos nuevos: `company_exists` y `key_limit_reached` → conflicto; `company_limit_reached` → límite (con `retry_after`).
- `Problem` trae `company` (en `company_exists`), `balance_cop` y `price_cop` (en `insufficient_balance` de una operación cobrada); `required_permission` admite los permisos de la llave de administración.
- La tabla de operaciones dice qué llave admite cada una (`llave`: `api_o_administracion`, `administracion` o `ninguna`).
- `trevol.NewClient` acepta la llave de administración (`tva_test_…`, `tva_live_…`) además de la de una empresa (`tvl_…`); el entorno sale
  del prefijo igual que antes y `c.KeyType` dice cuál es (`trevol.KeyTypeAPI` o `trevol.KeyTypeAccount`). Una llave mal formada sigue
  siendo `KindConfiguration`, con un mensaje que nombra los cuatro prefijos.
- `trevol.WithCompany("com_…")` y `c.ForCompany("com_…")` (un cliente para otra empresa que comparte el transporte y las conexiones):
  con una llave de administración, cada petición a una ruta de empresa lleva `Trevol-Company` y `Trevol-Environment`; sin empresa, la
  llamada falla con `KindConfiguration` antes de salir a la red. Con una llave `tvl_` esas cabeceras no se mandan nunca, pasar una empresa
  es `KindConfiguration` y una operación de `AccountAPI` también falla con `KindConfiguration` antes de salir a la red. `c.Company()` dice
  la empresa del cliente.
- Imprimir un `*trevol.Client` (con cualquier verbo de `fmt`) solo muestra el prefijo de la llave.

## [0.2.0] — 2026-11-18

Regenerado con la API de noviembre de 2026 (versión `2026-11-18`, la de los errores juntos; 149 operaciones y 70 tipos de evento).

- Recursos nuevos: `CustomersAPI`, `DianDocumentsAPI`, `DianSyncsAPI`, `InvoiceBatchesAPI`, `NumberingAssociationsAPI`, `NumberingRangesAPI`, `NumberingRequestsAPI`, `PaperNumberingRangesAPI`, `ProductsAPI`, `RadianStatusAPI`, `ReportRunsAPI`, `ReportsAPI`: lo recibido y el XML de la DIAN, la sincronización, el estado RADIAN, clientes y productos, numeraciones
  (también el modo simulado de las llaves de prueba), lotes de facturas, reportes (también los asíncronos) y la numeración de talonario.
- `Trevol-Version` por defecto: `2026-11-18`.
- Errores tipados para los códigos nuevos (`company_busy`, `confirmation_required`, `duplicate_customer`, `duplicate_product`,
  `numbering_request_in_progress`, `range_protected`, `paper_numbering_limit_reached` → conflicto; `automatic_mode_unavailable`,
  `portal_read_required`, `report_too_large` → precondición).
- Las líneas de factura, notas y documento equivalente aceptan la línea completa o la del catálogo (`product_id`) en un mismo modelo.
- La exportación de documentos (`exportDocuments`) se descarga como archivo (CSV o Excel).
- `Acquirer` trae `personal_data` (`redacted` si la llave no tiene `customers:read`: `email` viene en null). El PDF, el XML y el
  ZIP de lo emitido y el archivo de un reporte asíncrono con datos personales responden 403 `permission_denied` sin ese permiso.

## [0.1.0] — 2026-11-01

Primera versión (API `2026-10-22`).

- `trevol.NewClient` con un campo por etiqueta de la API (`InvoicesAPI`, `CreditNotesAPI`, `DebitNotesAPI`, `SupportDocumentsAPI`,
  `PayrollsAPI`, `RadianEventsAPI`, `WebhookEndpointsAPI`, `EventsAPI`, `RequestsAPI`, `BalanceAPI`, `GeneralAPI`…) y los modelos
  generados de la especificación (paquete `generado`).
- `Idempotency-Key` automática (la misma en los reintentos), reintentos con espera creciente que respetan `Retry-After`, `*trevol.Error` con
  una clase (`Kind…`) por `code`, entorno deducido del prefijo de la llave, `Trevol-Version` fijada.
- `trevol.Paginate` (`iter.Seq2`), descargas de PDF, XML y ZIP como `*os.File`, `trevol.NewWebhook(...).Verify` y `trevol.EventData`
  (Standard Webhooks).
