# Sup3API asset module (Sup3API)

An additive 3D asset API extension to [Sub2API](https://github.com/Wei-Shaw/sub2api).
The upstream Go module, LLM APIs, administrator UI and license are preserved.
The customer portal is independently branded Sup3API. This fork adds an API
module; it does not put game/editor workflows into the gateway. See [portal docs](docs/SUP3API.md).

## What works

| Unified operation | Tripo V3 | Meshy |
| --- | --- | --- |
| `text_to_3d` | single task | preview + refine inside one job |
| `image_to_3d` | one image | one image |
| `multi_image_to_3d` | four slots: front/left/back/right, >=2 populated | 1–4 images |
| `retexture` | texture task | retexture task |
| `rig` | versioned rigging | humanoid rigging |
| `animate` | preset names | numeric action IDs represented as strings |

Outputs include original GLB/FBX, geometry GLB, material/PBR manifests, embedded
texture images, skeleton JSON, per-vertex skin-weight JSON and animation JSON.
Every derived file records its source artifact. Missing rigs are `absent`;
compressed/unsupported accessors are explicitly `unsupported`.

## Run

Use the normal upstream PostgreSQL/Redis setup, then enable the module:

```dotenv
SUP3_ENABLED=true
SUP3_ALLOWED_USER_IDS=1
SUP3_DATA_DIR=/app/data/sup3-assets
```

`SUP3_ALLOWED_USER_IDS` is mandatory: only these host users can spend the
operator's provider credits. Generate a normal Sup3API API key for that user.
Keys must be enabled and unexpired for every asset request. LLM USD quota exhaustion
does not consume or disable the separate provider-credit allowance. Job reads are also restricted
to the API key that created them; revoking a key blocks access.

Build with the upstream Dockerfile, and use the additive Compose override:

```sh
docker build -t sup3api:local .
# Configure deploy/.env using deploy/.env.example and add the SUP3_* values.
docker compose --env-file deploy/.env -f deploy/docker-compose.yml -f deploy/sup3.compose.yml up -d
```

Or build the backend natively (a backend-only build has no embedded web UI):

```sh
cd backend
go build -o sup3api ./cmd/server
```

The module defaults to disabled. Add Tripo/Meshy API-key accounts in administrator
account management. Application keys remain separate. Provider keys are not returned
to applications. Existing tasks pin their upstream account and credential revision;
disabling/replacing credentials blocks further upstream calls rather than failing over. `APIKeys.txt`, `.local/` and `sup3-data/` are ignored.
The local test server created during development runs on `127.0.0.1:18763`.
Its private launch/configuration files are in `.local/` and are not part of the fork.

## Unified API

All endpoints use `Authorization: Bearer <Sup3API API key>`.

```sh
curl "$BASE_URL/v1/assets/jobs" \
  -H "Authorization: Bearer $SUP3_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: example-explorer-001' \
  -d '{
    "provider":"meshy",
    "operation":"text_to_3d",
    "model":"meshy-7.1",
    "inputs":{"prompt":"A stylized explorer in T pose"},
    "parameters":{"texture":true,"pbr":true,"target_faces":10000}
  }'
```

Poll `GET /v1/assets/jobs/{id}`. `status=succeeded` means the provider finished;
`delivery_status=ready` means local files and extracted components are ready.
Artifact URLs are relative, authenticated downloads on this gateway, with SHA-256
and byte size. They do not expose expiring upstream links.

| Endpoint | Purpose |
| --- | --- |
| `GET /v1/assets/capabilities` | provider availability and capabilities |
| `POST /v1/assets/quotes` | validate/normalize request; official native-credit estimate |
| `POST /v1/assets/jobs` | create a durable asynchronous job; `Idempotency-Key` required |
| `GET /v1/assets/jobs` | latest 100 jobs for the authenticated owner/key |
| `GET /v1/assets/jobs/{id}` | execution, usage, delivery and artifact manifest |
| `GET /v1/assets/jobs/{id}/artifacts/{artifact_id}` | protected file download |
| `POST /v1/assets/jobs/{id}/cancel` | local queued cancellation; Meshy pending cancellation |
| `POST /v1/assets/jobs/{id}/retry-delivery` | retry failed delivery without regeneration |
| `POST /v1/assets/jobs/{id}/refresh-artifacts` | refresh result URLs and re-extract completed results |

Rig/retexture use `inputs.job_id` or a public HTTPS `inputs.model_url`.
Animation requires a successful `rig` job ID. Examples:

```json
{"provider":"meshy","operation":"rig","inputs":{"job_id":"job_..."},"parameters":{}}
```
```json
{"provider":"tripo","operation":"animate","inputs":{"job_id":"job_..."},"parameters":{"animations":["preset:biped:walk"]}}
```

Tripo image/model inputs use public HTTPS URLs. Meshy image inputs additionally
accept validated PNG/JPEG Base64 data URIs (10 MiB decoded/image, 16 MiB JSON body).
Binary uploads and cross-provider job references are not supported.
Native field envelopes and full per-step provider results are documented in
[Sup3API API](docs/SUP3API.md). Provider-specific controls
belong in validated `provider_options`; unsupported combinations fail instead of
being silently ignored. Tripo `max_faces` is a bound, Meshy `target_faces` a target.
Do not interchange them. Tripo rig v1 uses `preset:biped:*`; v2.5 uses its own preset
names. See [OpenAPI](docs/sup3/openapi.json) for types and [architecture](docs/sup3/ARCHITECTURE.md).

## Billing and durability

Quotes use the official price snapshot dated 2026-09-29, multiplier **1**.
`cost.kind=reported` uses upstream consumed credits; partial or missing usage is
never presented as a final billed amount. Tripo publishes $0.01/credit. Meshy has
no universal verified dollar conversion here, so its USD field is omitted.

Credits are paid directly by the operator's upstream API account. This module does
**not** debit Sub2API's LLM USD wallet or inherit its dollar-denominated spending
limits. It is an allowlisted provider-credit gateway, not a USD resale billing
implementation. Do not enable it for untrusted paying tenants without adding
per-user credit budgets/reservations. Studio subscription credits are not assumed
to fund the Tripo API; official API keys are used, not browser sessions or OAuth.

Jobs and private provider metadata live in the independent `sup3_jobs` PostgreSQL
table. Polling resumes after restart. Workers use leases and fencing tokens.
The intent is committed before each paid POST: ambiguous submission or a crash
at that boundary yields `submission_unknown`, never an automatic paid retry.
Reconcile that job's upstream account manually before making a replacement.
Delivery retries only query/download existing results. CDN URL refresh is subject
to the provider's retention period. Multi-instance deployments need shared storage
at the same `SUP3_DATA_DIR` path. Back up that directory and PostgreSQL together.

GLB component extraction handles interleaved, normalized and sparse accessors.
It preserves glTF node/skin/mesh/primitive indices and coordinate conventions.
Geometry derivatives strip material and rig bindings while retaining original BIN
bytes; they are not byte-minimized or anonymized exports. Draco/meshopt decoding,
FBX component extraction and automatic engine import are not implemented.
Downloads are bounded at 256 MiB/file and block private destinations and redirects.

## Tests and upstream upgrades

```sh
cd backend
go test -tags unit ./internal/sup3 ./internal/server/middleware ./internal/server/routes
# Use a dedicated PostgreSQL database; tests create/drop isolated schemas only.
SUP3_TEST_DSN='host=127.0.0.1 dbname=sup3_test user=... password=... sslmode=disable' go test ./internal/sup3 -count=1
go vet -tags unit ./internal/sup3 ./internal/server/middleware ./internal/server/routes
go build ./cmd/server
```

The offline suite covers prices, provider envelopes, preview/refine, ambiguous
submission, persistence/restarts, owner/key isolation, idempotency, GLB extraction
and unsafe URL rejection. Live tests use real provider credits; see the separate
[verification report](docs/sup3/VERIFICATION.md) for the exact executed coverage.

`tools/sup3-smoke.py --provider meshy --run-id my-test` requests a quote only.
Set `SUP3_API_KEY` to your gateway key; add `--paid` to run the generation/rig/animation
pipeline and verify every artifact hash. Reuse the same run ID to resume a run.

Upstream integration is limited to one route-registration call and an optional
identity-only mode in the existing auth middleware. No Ent/Wire/generated schemas
or provider enums are modified. Rebase with `git fetch upstream` then
`git rebase upstream/main`, inspect those two integration points, and rerun the
commands above. Do not use the upstream binary updater on this fork: it would
replace the extended binary with upstream Sub2API. Build this fork for upgrades.

Sub2API's existing LGPL-3.0 license and notices remain in place.

## Unified output and native compatibility

New requests support `output.formats`, `output.required_components` and validated `extensions.<provider>`. Operation capabilities describe format conditions and extension schemas. Existing field envelopes remain supported. Native `/providers/meshy` and `/providers/tripo/v3` endpoints preserve covered provider workflows and response bodies, with scoped task ownership and optional durable idempotency. Native tasks/files are separate from unified jobs/delivery and use provider credits. No upload, lists, webhooks or historical task import. See [compatibility contract](docs/COMPATIBILITY.md) and on-site `/docs/compatibility` / `/docs/output`.
