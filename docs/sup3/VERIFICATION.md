# Sup3API verification — 2026-09-29

Upstream base: `Wei-Shaw/sub2api` at
`9a62841fd124d026cf3694fcf9b79e98addcdbdc`.
Environment: Windows native Go 1.27.1, PostgreSQL 14 and Redis 6 in WSL.
The full fork server ran on `127.0.0.1:18763` in standard mode. Requests used a
normal Sub2API API key and the new `/v1/assets` routes, not a standalone mock gateway.

## Live provider calls

All 12 unified jobs succeeded and reached `delivery_status=ready`.
Meshy text generation included two real upstream tasks (preview and refine).

| Operation | Tripo quoted / reported credits | Meshy quoted / reported credits |
| --- | ---: | ---: |
| Text to 3D + PBR | 20 / 20 | 30 / 30 |
| Image to 3D + PBR | 30 / 30 | 30 / 30 |
| Multiple images to 3D + PBR | 30 / 30 | 30 / 30 |
| Retexture + PBR | 10 / 10 | 10 / 10 |
| Rig | 25 / 25 | 5 / 5 |
| Animation, one action | 10 / 10 | 3 / 3 |
| **Total per provider** | **125 / 125** | **108 / 108** |

Generation models: Tripo `v3.1-20260211`, Meshy `meshy-7.1`.
Retexture models: Tripo `v3.5-20260815`, Meshy `meshy-7`.
Tripo humanoid rig: `v1.0-20240301`; animation: `preset:biped:walk`.
Meshy animation: action ID `0`.

The Tripo console exposed a free 14-day API wallet. Claiming it provided 600
credits; the final API/console balance was **475**, with zero frozen credits.
Meshy's balance changed from **1100** to **992**. Both supplied API keys were valid;
neither needed recreation. Tripo Studio subscription credits were not used.
Meshy's browser Google login hit a FedCM network error in the in-app browser;
its official API authentication and the real tasks succeeded independently.

Image inputs used the generated Meshy character thumbnail. The multiview smoke
test repeated that same view in two slots (Tripo `[front,left,"",""]`, Meshy two
URLs). This verifies serialization, provider execution and delivery, **not**
multi-angle reconstruction quality. Visual quality and every optional model/
resolution/topology combination are outside this test's claims.

## Files and components

Downloaded and verified **117 artifacts**, totaling **322,058,537 bytes**.
Every file's authenticated HTTP response matched its manifest's byte size and
SHA-256. Generated source files and credentials are local ignored data, not Git content.

| Real rig output | Joints | Vertices with extracted weights |
| --- | ---: | ---: |
| Tripo | 41 | 6,322 |
| Meshy | 24 | 13,693 |

Checks included joint indices within skin bounds, inverse-bind-matrix counts,
weights in range and normalized sums, and exported vertex counts matching the
source GLB POSITION accessors. Material manifests referenced existing image
artifacts. Geometry derivatives had material/skin bindings removed while original
GLBs remained intact. Unrigged generation outputs correctly reported no skeleton
or skin weights. Both animation jobs yielded extracted animation clips, skeletons
and weights. Meshy's rig GLB also contained one basic animation clip.

Refreshing completed animation artifacts reran downloads/extraction without
creating new generation tasks or changing consumed credits. This covered the
Meshy distinction between a GLB tagged as `animation` and one tagged as `model`.

## Automated and integration checks

- `go test -tags unit ./internal/sup3 ./internal/server/middleware ./internal/server/routes`
- Sup3 tests with `SUP3_TEST_DSN` set: actual PostgreSQL isolated schemas; concurrent
  idempotency, hash conflicts, owner/key isolation, restart between preview/refine,
  stale-worker fencing, and crash/ambiguous POST recovery without paid replay.
- GLB fixtures: interleaved and normalized weights, sparse accessors, bounds checks,
  original-file preservation and animation-container component extraction.
- Provider contract/price validation, multiview slot mapping, and prevention of
  deleting a completed Meshy task via the cancellation endpoint.
- Host auth unit tests: identity-only mode accepts zero LLM wallet balance while
  ordinary LLM auth still rejects it; revoked keys and disabled users remain blocked.
- Actual HTTP negative checks: missing authentication, other-key job/artifact
  access, cross-key source-job references, conflicting idempotency bodies and
  revoked credentials. Same-body idempotent replay returned the existing job.
- OpenAPI 3.1 specification validation; all 12 real job responses validated against
  the published Job schema.
- `go vet -tags unit` on the changed Sup3/auth/route packages and a full
  `go build ./cmd/server`.

No frontend changes were made. Docker deployment and a Docker frontend-embedded
build were not run on this machine. No assertion is made that Sub2API's entire
unrelated test suite or all provider option combinations were exercised.

## Deliberate boundaries

This is an operator-provisioned gateway using native provider credits. It does not
implement dollar resale billing, per-tenant credit budgets, a provider-account UI,
Studio OAuth, file-upload hosting, webhook delivery or a Hunyuan adapter.
The provider interface and versioned artifact graph provide extension points.
Draco/meshopt and FBX component decoding are reported as unsupported, not fabricated.
See the main Sup3 README for supported API routes, deployment and upgrade boundaries.
