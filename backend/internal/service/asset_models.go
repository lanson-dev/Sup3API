package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"time"
)

const meshyModelSpec = "https://docs.meshy.ai/openapi.json"

var tripoModelDocs = []string{
	"https://developers.tripo3d.ai/en/docs/generation-text-to-model/standard",
	"https://developers.tripo3d.ai/en/docs/models-texture",
	"https://developers.tripo3d.ai/en/docs/animations-rig",
}
var tripoModelVersion = regexp.MustCompile(`v[0-9]+\.[0-9]+-[0-9]{8}`)

// Neither provider publishes an authenticated /models endpoint. Synchronize the
// official contract instead, and label the source so this is not an entitlement claim.
func (s *AccountTestService) syncAssetModelCatalog(ctx context.Context, account *Account) (*UpstreamModelCatalog, error) {
	if err := ValidateAssetAccount(account); err != nil {
		return nil, newUpstreamModelSyncConfigError("Invalid asset account", err)
	}
	if s.httpUpstream == nil {
		return nil, newUpstreamModelSyncConfigError("Upstream HTTP client is not configured", nil)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	catalog := &UpstreamModelCatalog{Source: "official_documentation", Metadata: map[string]UpstreamModelMetadata{}}
	urls := tripoModelDocs
	if account.Platform == "meshy" {
		urls = []string{meshyModelSpec}
	}
	for _, source := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}
		// Public documentation requests must never carry the account API key.
		resp, err := s.doUpstreamModelsRequest(req, "", account)
		if err != nil {
			return nil, newUpstreamModelSyncUpstreamError("Cannot fetch official model documentation", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, upstreamModelsBodyLimit+1))
		resp.Body.Close()
		if readErr != nil || resp.StatusCode != http.StatusOK || int64(len(body)) > upstreamModelsBodyLimit {
			return nil, newUpstreamModelSyncUpstreamError("Official model documentation is unavailable", readErr)
		}
		if account.Platform == "meshy" {
			var spec struct {
				Components struct {
					Schemas map[string]struct {
						Properties map[string]json.RawMessage `json:"properties"`
					} `json:"schemas"`
				} `json:"components"`
			}
			if err := json.Unmarshal(body, &spec); err != nil {
				return nil, newUpstreamModelSyncUpstreamError("Invalid official model specification", err)
			}
			for schema, operation := range map[string]string{"TextTo3DRequest": "text_to_3d", "ImageTo3DRequest": "image_to_3d", "MultiImageTo3DRequest": "multi_image_to_3d", "RetextureRequest": "retexture"} {
				var model struct {
					Enum []string `json:"enum"`
				}
				if err := json.Unmarshal(spec.Components.Schemas[schema].Properties["ai_model"], &model); err != nil {
					return nil, newUpstreamModelSyncUpstreamError("Invalid model enum for "+schema, err)
				}
				ids := model.Enum
				if len(ids) == 0 {
					return nil, newUpstreamModelSyncUpstreamError("Official specification has no model enum for "+schema, nil)
				}
				for _, id := range ids {
					metadata := catalog.Metadata[id]
					metadata.ID = id
					metadata.OutputModalities = []string{"3d"}
					metadata.Operations = append(metadata.Operations, operation)
					catalog.Metadata[id] = metadata
				}
			}
		} else {
			ids := tripoModelVersion.FindAllString(string(body), -1)
			if len(ids) == 0 {
				return nil, newUpstreamModelSyncUpstreamError("Official documentation has no model versions; previous snapshot preserved", nil)
			}
			for _, id := range ids {
				catalog.Metadata[id] = UpstreamModelMetadata{ID: id, OutputModalities: []string{"3d"}}
			}
		}
	}
	for id, metadata := range catalog.Metadata {
		sort.Strings(metadata.Operations)
		catalog.Metadata[id] = metadata
		catalog.Models = append(catalog.Models, id)
	}
	sort.Strings(catalog.Models)
	if len(catalog.Models) == 0 {
		return nil, newUpstreamModelSyncUpstreamError("Official documentation returned no models", nil)
	}
	catalog.Sources = urls
	if account.ID > 0 {
		if s.accountRepo == nil {
			return nil, newUpstreamModelSyncInternalError("Account repository is not configured", nil)
		}
		snapshot := UpstreamModelMetadataSnapshot{Source: catalog.Source, SyncedAt: time.Now().UTC().Format(time.RFC3339), Models: catalog.Metadata}
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{UpstreamModelMetadataExtraKey: snapshot}); err != nil {
			return nil, newUpstreamModelSyncInternalError("Cannot save model catalog", fmt.Errorf("account %d: %w", account.ID, err))
		}
		account.SetUpstreamModelMetadataSnapshot(snapshot)
	}
	return catalog, nil
}
