package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

type assetCatalogHTTP struct {
	HTTPUpstream
	body string
}

func (h *assetCatalogHTTP) DoWithTLS(r *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	if r.Header.Get("Authorization") != "" {
		panic("provider secret sent to public documentation")
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(h.body))}, nil
}
func TestAssetModelCatalogSync(t *testing.T) {
	spec := `{"components":{"schemas":{`
	for i, name := range []string{"TextTo3DRequest", "ImageTo3DRequest", "MultiImageTo3DRequest", "RetextureRequest"} {
		if i > 0 {
			spec += ","
		}
		spec += `"` + name + `":{"properties":{"unrelated_option":{"enum":[1,2]},"ai_model":{"enum":["meshy-7.1","latest"]}}}`
	}
	spec += `}}}`
	for _, platform := range []string{"tripo", "meshy"} {
		t.Run(platform, func(t *testing.T) {
			body := spec
			if platform == "tripo" {
				body = `<main><code>v3.1-20260211</code><code>v2.5-20250123</code></main>`
			}
			repo := &upstreamModelMetadataRepoStub{}
			svc := &AccountTestService{accountRepo: repo, httpUpstream: &assetCatalogHTTP{body: body}}
			account := &Account{ID: 8, Platform: platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "private-test-key"}}
			catalog, err := svc.SyncUpstreamModelCatalog(context.Background(), account)
			require.NoError(t, err)
			require.Len(t, catalog.Models, 2)
			require.Equal(t, "official_documentation", catalog.Source)
			require.Equal(t, int64(8), repo.accountID)
			require.NotNil(t, account.GetUpstreamModelMetadataSnapshot())
			require.NotContains(t, catalog.Models, "claude-sonnet-4-5")
			// Invalid/changed documents must not overwrite a previous snapshot.
			repo.updates = nil
			svc.httpUpstream = &assetCatalogHTTP{body: `<html>Not found</html>`}
			_, err = svc.SyncUpstreamModelCatalog(context.Background(), account)
			require.Error(t, err)
			require.Nil(t, repo.updates)
		})
	}
}
