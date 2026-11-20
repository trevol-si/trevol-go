# Registro de cambios

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el versionado, [SemVer](https://semver.org/lang/es/): antes
de la 1.0, un cambio incompatible sube el número menor.

## [0.4.0] — 2026-11-19

Regenerado con las novedades de la API de noviembre (la misma versión `2026-11-18`; 178 operaciones y 70 tipos de evento). Todo es
aditivo: el código que ya funciona no cambia.

- Recursos nuevos `CatalogsAPI`, `PdfTemplatesAPI`, `PdfDesignsAPI` y `CompanySettingsAPI`: las listas de valores (`listCatalogs` y `listCatalogEntries`: los códigos que acepta cada campo), las
  plantillas y los diseños del PDF (`listPdfTemplates`, `listPdfDesigns`, `createPdfDesign`, `retrievePdfDesign`, `updatePdfDesign` y
  `archivePdfDesign`) y los ajustes del PDF y el logo de la empresa (`retrievePdfSettings`, `updatePdfSettings`, `retrieveCompanyLogo`,
  `updateCompanyLogo`, `deleteCompanyLogo` y `retrieveCompanyLogoFile`).
- Reenviar el correo de un documento validado: `sendInvoiceEmail`, `sendCreditNoteEmail`, `sendDebitNoteEmail`,
  `sendEquivalentDocumentEmail` y `sendEquivalentDocumentAdjustmentEmail` (con `Idempotency-Key`; `to` es opcional). El documento trae
  `delivery.email_resends`.
- Buscar por número: las listas de documentos y `listDianDocuments` aceptan `number` y `prefix`.
- La factura, su vista previa y cada factura de un lote aceptan `numbering` (`prefix`) para elegir la numeración; la vista previa la
  devuelve en `numbering`.
- `pdf_design_id` al crear o corregir facturas, notas y documentos equivalentes POS; esos documentos traen `pdf_design_id` y
  `pdf_design`.
- `simulated` en los documentos, el lote, la empresa y los eventos de la empresa simulada.
- Webhooks: `follow_latest_catalog` en los endpoints.
- Errores tipados para los códigos nuevos: `logo_invalid` y `pdf_design_invalid` → `KindValidation`; `pdf_design_limit_reached`,
  `pdf_design_in_use`, `pdf_design_outdated` y `pdf_design_archived` → `KindConflict`. `Problem` trae `findings` (en `pdf_design_invalid`) y
  `current_version` (en `pdf_design_outdated`).

## [0.3.1] — 2026-11-19

Solo textos de la especificación (la misma versión `2026-11-18` de la API; nada cambia en el código):

- Crear una llave de empresa con la llave de administración: ya no dice que una llave de escritura de producción se crea en la app.
- `createDianSync`: los topes dicen 2 sincronizaciones de la cuenta en curso a la vez (no 5).
- Los eventos de un endpoint de webhooks: 69 tipos en el catálogo (decía 60).
- `api_version` de un evento: la versión del formato de los eventos (hoy `2026-10-22`), la misma para todas las llaves.
- `stop_reason` de la habilitación de una empresa (`AccountCompanyOnboarding`) lista un valor conocido menos: esa parada llega ahora como `dian_portal_unavailable`.
- `GET /v1/balance`: el ejemplo trae los precios vigentes de `operation_prices` (200, 50, 100 y 500 pesos) y la descripción dice solo que 0 es gratis.

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
