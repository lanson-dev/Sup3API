package service

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/sup3"
	"github.com/stretchr/testify/require"
)

type assetAccountAdmin struct {
	AdminService
	accounts []Account
	group    *Group
}

func (a *assetAccountAdmin) ListAccountsForSchedulerScoreFilter(context.Context, string, string, string, string, int64, string) ([]Account, error) {
	return append([]Account(nil), a.accounts...), nil
}

func (a *assetAccountAdmin) GetGroup(context.Context, int64) (*Group, error) {
	if a.group != nil {
		return a.group, nil
	}
	return &Group{ID: 7, Platform: "meshy", Status: StatusActive}, nil
}

func TestAssetAccountSelectionAndCredentialPinning(t *testing.T) {
	a := &assetAccountAdmin{accounts: []Account{
		{ID: 1, GroupIDs: []int64{7}, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 10, Credentials: map[string]any{"api_key": "first"}},
		{ID: 2, GroupIDs: []int64{7}, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 20, Credentials: map[string]any{"api_key": "second"}},
	}}
	ctx := sup3.WithRouting(context.Background(), 7)
	p, binding, err := ResolveAssetAccount(ctx, a, "meshy", "")
	require.NoError(t, err)
	require.Equal(t, "first", p.(*sup3.RemoteProvider).Key)
	a.accounts[1].Priority = 0
	p, _, err = ResolveAssetAccount(ctx, a, "meshy", binding)
	require.NoError(t, err)
	require.Equal(t, "first", p.(*sup3.RemoteProvider).Key)
	a.accounts[0].Status = "inactive"
	_, _, err = ResolveAssetAccount(ctx, a, "meshy", binding)
	require.Error(t, err, "disabled tasks must not fail over")
	a.accounts[0].Status = StatusActive
	a.accounts[0].Credentials["api_key"] = "rotated"
	_, _, err = ResolveAssetAccount(ctx, a, "meshy", binding)
	require.Error(t, err, "credential replacement must not redirect old tasks")
	a.accounts[0].Schedulable = false
	expired := time.Now().Add(-time.Hour)
	a.accounts[1].ExpiresAt = &expired
	a.accounts[1].AutoPauseOnExpired = true
	_, _, err = ResolveAssetAccount(ctx, a, "meshy", "")
	require.Error(t, err)
}

func TestAssetAccountValidation(t *testing.T) {
	for _, platform := range []string{"tripo", "meshy"} {
		a := &Account{Platform: platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "upstream-secret"}}
		require.NoError(t, ValidateAssetAccount(a))
		a.Type = AccountTypeOAuth
		require.Error(t, ValidateAssetAccount(a))
		a.Type = AccountTypeAPIKey
		a.Credentials["base_url"] = "https://attacker.example"
		require.Error(t, ValidateAssetAccount(a))
		delete(a.Credentials, "base_url")
		a.Credentials["api_key"] = ""
		require.Error(t, ValidateAssetAccount(a))
	}
}

func TestAssetGroupIsolation(t *testing.T) {
	a := &assetAccountAdmin{accounts: []Account{
		{ID: 1, GroupIDs: []int64{8}, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 0, Credentials: map[string]any{"api_key": "other-group"}},
		{ID: 2, GroupIDs: []int64{7}, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 10, Credentials: map[string]any{"api_key": "own-group", "model_mapping": map[string]any{"asset-fast": "meshy-6"}}},
	}}
	ctx := sup3.WithRouting(context.Background(), 7)
	p, binding, err := ResolveAssetAccount(ctx, a, "meshy", "")
	require.NoError(t, err)
	require.Equal(t, "own-group", p.(*sup3.RemoteProvider).Key)
	require.Equal(t, "meshy-6", p.(*sup3.RemoteProvider).ResolveModel("asset-fast"))
	_, _, err = ResolveAssetAccount(sup3.WithRouting(ctx, 8), a, "tripo", "")
	require.Error(t, err)
	_, _, err = ResolveAssetAccount(context.Background(), a, "meshy", "")
	require.Error(t, err)
	// Internal workers still resolve the submitted task after group changes.
	_, _, err = ResolveAssetAccount(context.Background(), a, "meshy", binding)
	require.NoError(t, err)
	a.accounts[1].GroupIDs = nil
	_, _, err = ResolveAssetAccount(ctx, a, "meshy", "")
	require.Error(t, err)
	a.group = &Group{ID: 7, Platform: "meshy", Status: "inactive"}
	_, _, err = ResolveAssetAccount(ctx, a, "meshy", binding)
	require.Error(t, err)
}

func TestAssetQuotesEnforceGroupAndAccountModels(t *testing.T) {
	a := &assetAccountAdmin{group: &Group{ID: 7, Platform: "meshy", Status: StatusActive, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"asset-fast"}}}, accounts: []Account{{ID: 1, GroupIDs: []int64{7}, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Credentials: map[string]any{"api_key": "private", "model_mapping": map[string]any{"asset-fast": "meshy-6"}}}}}
	e := sup3.NewEngine(nil, t.TempDir(), sup3.NewProvider("meshy", ""))
	e.ResolveProvider = func(ctx context.Context, p, b string) (sup3.Provider, string, error) {
		return ResolveAssetAccount(ctx, a, p, b)
	}
	quote := func(model string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/assets/quotes", strings.NewReader(`{"provider":"meshy","operation":"text_to_3d","model":"`+model+`","inputs":{"prompt":"crate"}}`))
		r = r.WithContext(sup3.WithRouting(r.Context(), 7))
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r, 1, 2)
		return w
	}
	w := quote("asset-fast")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"resolved_model":"meshy-6"`)
	require.Equal(t, 403, quote("meshy-7.1").Code)
	a.group.ModelAllowlist.Enabled = false
	require.Equal(t, 503, quote("meshy-7.1").Code, "account whitelist must still apply")
}
