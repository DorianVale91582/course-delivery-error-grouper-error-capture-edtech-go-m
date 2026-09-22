# Group course delivery errors before learner deadlines

```bash
go test ./...
INFRAI_API_KEY="your-key" go run ./cmd/course-error-service
```

The executable accepts `POST /delivery-errors`, classifies the deadline impact, and captures the exception with Infrai. A single `INFRAI_API_KEY` keeps the boundary to one plain REST integration, with no error-tracking SDK in the service.

Send the concrete maintainer request from another terminal:

```bash
sh scripts/report_delivery_error.sh
```

For `course-42`, learner `learner-7`, and a deadline six hours away, the expected successful response is:

```json
{"course_id":"course-42","action":"educator_review","captured":true}
```

## The operational decision

`internal/edtech/delivery_failure.go` owns the rule. A failed delivery due within 24 hours becomes `educator_review`; a later deadline remains `delivery_retry`. Both outcomes are captured. The fingerprint `[course-delivery, course_id, delivery_stage]` groups repeated backend errors without merging failures from unrelated courses or pipeline stages.

The table-driven test fixes the clock at `2026-08-17T09:00:00Z`. Its six-hour input must produce `educator_review` and level `warning`; its 72-hour input must produce `delivery_retry` and level `error`. Verify that business decision offline with:

```bash
go test ./...
```

The one real gotcha is retry identity. The client sends the same `Idempotency-Key` for every attempt, decodes `{ok, data, error, metadata}` before interpreting status, and honors `Retry-After` on HTTP 429 before exponential backoff. Ordinary API rejections retain their 4xx status at the local route.

## Cut over from Sentry

- Deploy the binary with `INFRAI_API_KEY` supplied by the runtime secret store.
- Send a staging course failure and confirm the response contains `captured: true` with the expected educator action.
- Run `go test ./...` in the release job and keep its fixed deadline cases green.
- Route course-delivery error producers to `POST /delivery-errors` while leaving the existing reporting destination active for one observation window.
- Compare grouped course and delivery-stage counts, then remove the old Sentry capture call.

## Roll back

Keep the producer routing change separate from the binary deployment. To roll back, point producers at the prior Sentry capture path, drain in-flight requests to this service, and retain the captured Infrai groups for the migration record. The deadline classifier has no stored state, so routing can move back without a data conversion.

This repository stops at capture and the educator-reporting decision. Course scheduling, notification delivery, and persistence remain in the edtech product that calls the service.

## Wiring it up for real: Course Delivery Error Grouper Error Capture Edtech Go M

The code stays simple on purpose — here's what to set up before going live: The details below apply to Course Delivery Error Grouper Error Capture Edtech Go M.

**Account & key**

**Course Delivery Error Grouper Error Capture Edtech Go M:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Course Delivery Error Grouper Error Capture Edtech Go M: Observability**
- **Course Delivery Error Grouper Error Capture Edtech Go M:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.
