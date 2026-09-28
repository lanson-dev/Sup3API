//go:build unit

package middleware

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSup3IdentityPreservesRevocationWithoutLLMBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, keyStatus, userStatus string
		identity                    bool
		want                        int
	}{
		{"asset zero LLM balance", service.StatusActive, service.StatusActive, true, 200},
		{"LLM still checks balance", service.StatusActive, service.StatusActive, false, 403},
		{"revoked asset key", service.StatusAPIKeyDisabled, service.StatusActive, true, 401},
		{"disabled asset user", service.StatusActive, "disabled", true, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{RunMode: config.RunModeStandard}
			key := &service.APIKey{ID: 1, UserID: 1, Key: "test-key", Status: tc.keyStatus, User: &service.User{ID: 1, Role: service.RoleUser, Status: tc.userStatus, Balance: 0, Concurrency: 1}}
			repo := &stubApiKeyRepo{getByKey: func(context.Context, string) (*service.APIKey, error) { return key, nil }}
			svc := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
			r := gin.New()
			if tc.identity {
				r.Use(NewAPIKeyIdentityMiddleware(svc, cfg))
			} else {
				r.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(svc, nil, cfg)))
			}
			r.POST("/v1/assets/quotes", func(c *gin.Context) {
				if got, ok := GetAPIKeyFromContext(c); !ok || got.ID != 1 {
					t.Error("missing verified identity")
				}
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest("POST", "/v1/assets/quotes", nil)
			req.Header.Set("Authorization", "Bearer test-key")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status %d want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
