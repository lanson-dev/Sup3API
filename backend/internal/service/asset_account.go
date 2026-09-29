package service

import (
	"context"
	"net/http"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/sup3"
)

func IsAssetPlatform(platform string) bool { return platform == "tripo" || platform == "meshy" }

func ValidateAssetAccount(a *Account) error {
	if !IsAssetPlatform(a.Platform) {
		return nil
	}
	key, _ := a.Credentials["api_key"].(string)
	base, _ := a.Credentials["base_url"].(string)
	if a.Type != AccountTypeAPIKey || strings.TrimSpace(key) == "" || key != strings.TrimSpace(key) || strings.ContainsAny(key, "\r\n") {
		return infraerrors.New(http.StatusBadRequest, "INVALID_ASSET_ACCOUNT", "Tripo/Meshy require an upstream API key")
	}
	// Official endpoints only. Do not silently accept a proxy/base URL that this
	// adapter cannot honour, or send credentials to an arbitrary host.
	if (base != "" && strings.TrimRight(base, "/") != sup3.NewProvider(a.Platform, "").BaseURL) || a.ProxyID != nil {
		return infraerrors.New(http.StatusBadRequest, "INVALID_ASSET_TRANSPORT", "asset accounts currently use the official endpoint without a proxy")
	}
	return nil
}

// ResolveAssetAccount reuses the administrator account store. Lower priority
// wins; equal priorities use ID order. No credential cache hides edits/disable.
func ResolveAssetAccount(ctx context.Context, admin AdminService, platform, binding string) (sup3.Provider, string, error) {
	if !IsAssetPlatform(platform) {
		return nil, "", &sup3.APIError{Code: "invalid_provider", Message: "unknown asset provider", HTTPStatus: 400}
	}
	accounts, err := admin.ListAccountsForSchedulerScoreFilter(ctx, platform, AccountTypeAPIKey, StatusActive, "", 0, "")
	if err != nil {
		return nil, "", err
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Priority != accounts[j].Priority {
			return accounts[i].Priority < accounts[j].Priority
		}
		return accounts[i].ID < accounts[j].ID
	})
	for i := range accounts {
		a := &accounts[i]
		if a.Platform != platform || a.Type != AccountTypeAPIKey || !a.IsSchedulable() || ValidateAssetAccount(a) != nil {
			continue
		}
		key, _ := a.Credentials["api_key"].(string)
		identity := sup3.AccountBinding(a.ID, key)
		if binding != "" && binding != identity && binding != sup3.CredentialFingerprint(key) {
			continue
		}
		// Preserve a legacy fingerprint for already persisted native tasks.
		if binding != "" {
			identity = binding
		}
		return sup3.NewProvider(platform, key), identity, nil
	}
	return nil, "", &sup3.APIError{Code: "provider_account_unavailable", Message: "configure or enable the original upstream account in administrator account management", HTTPStatus: 503, Retryable: true}
}
