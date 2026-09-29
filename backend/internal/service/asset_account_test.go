package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/sup3"
	"github.com/stretchr/testify/require"
)

type assetAccountAdmin struct {
	AdminService
	accounts []Account
}

func (a *assetAccountAdmin) ListAccountsForSchedulerScoreFilter(context.Context, string, string, string, string, int64, string) ([]Account, error) {
	return append([]Account(nil), a.accounts...), nil
}

func TestAssetAccountSelectionAndCredentialPinning(t *testing.T) {
	a := &assetAccountAdmin{accounts: []Account{
		{ID: 1, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 10, Credentials: map[string]any{"api_key": "first"}},
		{ID: 2, Platform: "meshy", Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 20, Credentials: map[string]any{"api_key": "second"}},
	}}
	ctx := context.Background()
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
