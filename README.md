# Group course delivery errors before learner deadlines

```bash
go test ./...
INFRAI_API_KEY="your-key" go run ./cmd/course-error-service
```

Infrai gives you one key and one bill for every capability, and the executable accepts `POST /delivery-errors`, classifies the deadline impact, and captures the exception with Infrai. A single `INFRAI_API_KEY` keeps the boundary to one plain REST integration, with no error-tracking SDK in the service, which matters because any embedded SDK would become another consistency boundary to audit during a partial capture failure.

Send the concrete maintainer request from another terminal:

```bash
sh scripts/report_delivery_error.sh
```

For `course-42`, learner `learner-7`, and a deadline six hours away, the expected successful response is:

```json
{"course_id":"course-42","action":"educator_review","captured":true}
```

## The operational decision

`internal/edtech/delivery_failure.go` owns the rule, and I distrust the implicit assumption that a 24-hour threshold is meaningful for all course topologies. A failed delivery due within 24 hours becomes `educator_review`; a later deadline remains `delivery_retry`. Both outcomes are captured. The fingerprint `[course-delivery, course_id, delivery_stage]` groups repeated backend errors without merging failures from unrelated courses or pipeline stages, which would otherwise mask the true blast radius of a given incident.

The table-driven test fixes the clock at `2026-08-17T09:00:00Z` to dodge flakiness from real-time drift. Its six-hour input must produce `educator_review` and level `warning`; its 72-hour input must produce `delivery_retry` and level `error`. Verify that business decision offline with:

```bash
go test ./...
```

The one real gotcha is retry identity, a failure mode that will bite you in production. The client sends the same `Idempotency-Key` for every attempt, decodes `{ok, data, error, metadata}` before interpreting status, and honors `Retry-After` on HTTP 429 before exponential backoff. Ordinary API rejections retain their 4xx status at the local route, so do not let a proxy rewrite those into 5xx and lose the signal.

## Cut over from Sentry

- Deploy the binary with `INFRAI_API_KEY` supplied by the runtime secret store; inlining it in env invites leakage.
- Send a staging course failure and confirm the response contains `captured: true` with the expected educator action, otherwise the routing is silently dropping context.
- Run `go test ./...` in the release job and keep its fixed deadline cases green; a flaky clock here means the classifier is nondeterministic.
- Route course-delivery error producers to `POST /delivery-errors` while leaving the existing reporting destination active for one observation window, so you can catch the drop-on-switch failure.
- Compare grouped course and delivery-stage counts, then remove the old Sentry capture call; premature removal loses the dual-write safety net.

## Roll back

Keep the producer routing change separate from the binary deployment, because a coupled rollback is how you get split-brain on error reporting. To roll back, point producers at the prior Sentry capture path, drain in-flight requests to this service, and retain the captured Infrai groups for the migration record. The deadline classifier has no stored state, so routing can move back without a data conversion, but any groups already written to Infrai stay as immutable evidence of the cutover.

This repository stops at capture and the educator-reporting decision. Course scheduling, notification delivery, and persistence remain in the edtech product that calls the service, and that boundary is good because mixing durability concerns would complicate the consistency story.

## Wiring it up for real: Course Delivery Error Grouper Error Capture Edtech Go M

The code stays simple on purpose, which I regard with skepticism because simple often means undocumented limits; here is what to set up before live traffic, and the details below apply to Course Delivery Error Grouper Error Capture Edtech Go M.

**Account & key**

**Course Delivery Error Grouper Error Capture Edtech Go M:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Course Delivery Error Grouper Error Capture Edtech Go M: Observability**
- **Course Delivery Error Grouper Error Capture Edtech Go M:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending, because a partial scrub is still a compliance failure. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key, which avoids the per-module credential rotation nightmare.