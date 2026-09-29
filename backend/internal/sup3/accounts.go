package sup3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// ProviderResolver selects a managed account when binding is empty, otherwise
// resolves exactly the original account and credential revision. Never fail over
// a submitted task to another upstream account.
type ProviderResolver func(context.Context, string, string) (Provider, string, error)

func CredentialFingerprint(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:])
}

func AccountBinding(id int64, key string) string {
	return strconv.FormatInt(id, 10) + ":" + CredentialFingerprint(key)
}

func (e *Engine) resolve(ctx context.Context, provider, binding string) (Provider, string, error) {
	if e.ResolveProvider != nil {
		return e.ResolveProvider(ctx, provider, binding)
	}
	p := e.Providers[provider]
	if p == nil || !p.Capability().Available {
		return nil, "", &APIError{Code: "provider_not_configured", Message: "no available upstream account", HTTPStatus: 503}
	}
	if remote, ok := p.(*RemoteProvider); ok {
		return p, CredentialFingerprint(remote.Key), nil
	}
	return p, binding, nil
}

func (e *Engine) jobProvider(ctx context.Context, j *Job) (Provider, error) {
	if e.ResolveProvider != nil && j.AccountBinding == "" {
		return nil, &APIError{Code: "legacy_account_unbound", Message: "legacy task requires upstream account reconciliation", HTTPStatus: 409}
	}
	p, _, err := e.resolve(ctx, j.Request.Provider, j.AccountBinding)
	return p, err
}
