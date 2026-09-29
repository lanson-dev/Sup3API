package service

import (
	"context"
	"net/http"
	"slices"
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
	routing := sup3.RoutingFromContext(ctx)
	if routing.GroupID > 0 {
		group, err := admin.GetGroup(ctx, routing.GroupID)
		if err != nil || group == nil || group.Status != StatusActive || (group.Platform != platform && group.Platform != PlatformComposite) {
			return nil, "", &sup3.APIError{Code: "asset_group_denied", Message: "API key group does not allow this provider", HTTPStatus: 403}
		}
		if routing.Model != "" && !group.ModelAllowlist.Allows(routing.Model) {
			return nil, "", &sup3.APIError{Code: "model_not_allowed", Message: "model is not allowed by the API key group", HTTPStatus: 403}
		}
	} else if binding == "" {
		return nil, "", &sup3.APIError{Code: "asset_group_required", Message: "bind the API key to a provider group", HTTPStatus: 403}
	}
	accounts, err := admin.ListAccountsForSchedulerScoreFilter(ctx, platform, AccountTypeAPIKey, StatusActive, "", routing.GroupID, "")
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
		if routing.GroupID > 0 && !slices.Contains(a.GroupIDs, routing.GroupID) {
			continue
		}
		if routing.Model != "" && !a.IsModelSupported(routing.Model) {
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
		provider := sup3.NewProvider(platform, key)
		provider.ResolveModel = a.GetMappedModel
		return provider, identity, nil
	}
	return nil, "", &sup3.APIError{Code: "provider_account_unavailable", Message: "configure or enable the original upstream account in administrator account management", HTTPStatus: 503, Retryable: true}
}
