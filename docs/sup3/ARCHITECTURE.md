# Sup3API: additive 3D API extension

Sup3API is a fork of Wei-Shaw/sub2api. Keep the upstream Go module name and existing
LLM providers intact. New code belongs in `backend/internal/sup3`; host integration
belongs in `backend/internal/server/routes/sup3.go`. Avoid generated Ent/Wire changes.
The host owns administrator upstream accounts and application API-key authentication.
Tripo/Meshy credentials live in account management, never in application key records. Sup3 owns 3D requests, provider adapters,
durable jobs, native-credit pricing, artifacts and portable component extraction.
Game rules, editors, scenes and DCC import bridges remain outside this API service.

## Contract decisions

- A job has a provider, pinned model, operation, typed inputs, normalized parameters
  and explicitly validated provider options. A job may have multiple upstream steps.
- Distinguish execution from artifact delivery. Never regenerate a paid model because
  a download failed. Resume polling known upstream IDs after restart. An uncertain
  submission must be reconciled instead of automatically resubmitted.
- Outputs are an artifact graph: original model/container, geometry, materials,
  texture maps, skeleton, skin weights, animations and previews. Formats are metadata,
  not artifact roles. A GLB can contain several roles; preserve the original.
- Extracting skeleton/weights means decoding actual glTF skins/accessors, including
  inverse bind matrices and vertex-to-joint mappings. Do not invent a rig for an
  unrigged asset. Report absent/unsupported components explicitly.
- Quotes and usage use **provider-native credits** with a default multiplier of 1.
  Tripo currently publishes USD 0.01/credit. Meshy plan credits do not establish a
  universal USD conversion; do not invent one. Keep these costs separate from the
  host's LLM USD wallet. The upstream key owner pays the provider directly.
- Authentication method is provider-specific. This implementation uses official API
  keys supplied by the operator. Studio browser sessions are not API keys and are
  not silently substituted when the official API has no credit.

## Provider research (2026-09-29)

| Provider | Execution | Outputs | Consequence |
| --- | --- | --- | --- |
| Tripo V3 | submit + unified task query; texture/rig/retarget operations | operation-specific model URLs, GLB/FBX, embedded PBR/rig | preserve steps and extract GLB components |
| Meshy | per-operation task routes; text preview then refine | model_urls, texture_urls; rigged GLB/FBX and animation results | one public job may contain preview/refine; preserve all files |
| Tencent Hunyuan3D | asynchronous submit/query, authenticated Tencent API | result file list, format/type metadata | adapter must support request signing and multiple output files |
| Hyper3D Rodin | asynchronous generation and result retrieval | multiple models/material files | no single model_url assumption |
| fal / Runware | asynchronous broker jobs | model-dependent or generic file lists | vendor and execution channel must remain distinguishable |

Sources: https://developers.tripo3d.ai/en/docs/authentication,
https://developers.tripo3d.ai/en/pricing,
https://docs.meshy.ai/api/pricing,
https://docs.meshy.ai/api/rigging,
https://cloud.tencent.com/document/product/1804,
https://developer.hyper3d.ai/.

## Verification requirements

1. Provider contract tests cover paths, envelopes, staged texturing, errors and prices.
2. Worker tests cover restart recovery, idempotency, ownership and uncertain submission.
3. Extraction tests verify actual vertex weights, skeleton hierarchy and PBR channels.
4. Live tests go through the unified HTTP API, download and inspect generated assets.
5. Preserve credentials outside Git; scan outgoing diff and tracked files before push.
6. Build the full fork and keep the upstream integration diff small enough to rebase.
