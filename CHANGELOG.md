# Registro de cambios

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el versionado, [SemVer](https://semver.org/lang/es/): antes
de la 1.0, un cambio incompatible sube el número menor.

## [Sin publicar]

Regenerado con la API de noviembre de 2026 (versión `2026-11-18`, la de los errores juntos; 149 operaciones y 70 tipos de evento).

- Recursos nuevos: `CustomersAPI`, `DianDocumentsAPI`, `DianSyncsAPI`, `InvoiceBatchesAPI`, `NumberingAssociationsAPI`, `NumberingRangesAPI`, `NumberingRequestsAPI`, `PaperNumberingRangesAPI`, `ProductsAPI`, `RadianStatusAPI`, `ReportRunsAPI`, `ReportsAPI`: lo recibido y el XML de la DIAN, la sincronización, el estado RADIAN, clientes y productos, numeraciones
  (también el modo simulado de las llaves de prueba), lotes de facturas, reportes (también los asíncronos) y la numeración de talonario.
- `Trevol-Version` por defecto: `2026-11-18`.
- Errores tipados para los códigos nuevos (`company_busy`, `confirmation_required`, `duplicate_customer`, `duplicate_product`,
  `numbering_request_in_progress`, `range_protected`, `paper_numbering_limit_reached` → conflicto; `automatic_mode_unavailable`,
  `portal_read_required`, `report_too_large` → precondición).
- Las líneas de factura, notas y documento equivalente aceptan la línea completa o la del catálogo (`product_id`) en un mismo modelo.
- La exportación de documentos (`exportDocuments`) se descarga como archivo (CSV o Excel).

## [0.1.0] — sin publicar

Primera versión (API `2026-10-22`).

- `trevol.NewClient` con un campo por etiqueta de la API (`InvoicesAPI`, `CreditNotesAPI`, `DebitNotesAPI`, `SupportDocumentsAPI`,
  `PayrollsAPI`, `RadianEventsAPI`, `WebhookEndpointsAPI`, `EventsAPI`, `RequestsAPI`, `BalanceAPI`, `GeneralAPI`…) y los modelos
  generados de la especificación (paquete `generado`).
- `Idempotency-Key` automática (la misma en los reintentos), reintentos con espera creciente que respetan `Retry-After`, `*trevol.Error` con
  una clase (`Kind…`) por `code`, entorno deducido del prefijo de la llave, `Trevol-Version` fijada.
- `trevol.Paginate` (`iter.Seq2`), descargas de PDF, XML y ZIP como `*os.File`, `trevol.NewWebhook(...).Verify` y `trevol.EventData`
  (Standard Webhooks).
