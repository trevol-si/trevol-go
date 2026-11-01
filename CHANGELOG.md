# Registro de cambios

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el versionado, [SemVer](https://semver.org/lang/es/): antes
de la 1.0, un cambio incompatible sube el número menor.

## [0.1.0] — sin publicar

Primera versión (API `2026-10-22`).

- `trevol.NewClient` con un campo por etiqueta de la API (`InvoicesAPI`, `CreditNotesAPI`, `DebitNotesAPI`, `SupportDocumentsAPI`,
  `PayrollsAPI`, `RadianEventsAPI`, `WebhookEndpointsAPI`, `EventsAPI`, `RequestsAPI`, `BalanceAPI`, `GeneralAPI`…) y los modelos
  generados de la especificación (paquete `generado`).
- `Idempotency-Key` automática (la misma en los reintentos), reintentos con espera creciente que respetan `Retry-After`, `*trevol.Error` con
  una clase (`Kind…`) por `code`, entorno deducido del prefijo de la llave, `Trevol-Version` fijada.
- `trevol.Paginate` (`iter.Seq2`), descargas de PDF, XML y ZIP como `*os.File`, `trevol.NewWebhook(...).Verify` y `trevol.EventData`
  (Standard Webhooks).
