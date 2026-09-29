package routes

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/sup3"
	"github.com/gin-gonic/gin"
)

// RegisterSup3Routes is the sole asset-module integration point. The module is
// disabled by default. Shared operator credentials require an explicit owner allowlist.
func RegisterSup3Routes(r *gin.Engine, keys *service.APIKeyService, cfg *config.Config, resolve sup3.ProviderResolver) {
	if os.Getenv("SUP3_ENABLED") != "true" {
		return
	}
	owners := map[int64]bool{}
	for _, s := range strings.Split(os.Getenv("SUP3_ALLOWED_USER_IDS"), ",") {
		if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && n > 0 {
			owners[n] = true
		}
	}
	if len(owners) == 0 {
		log.Fatal("SUP3_ENABLED requires SUP3_ALLOWED_USER_IDS; provider credits must not be shared with every gateway user")
	}
	dir := os.Getenv("SUP3_DATA_DIR")
	if dir == "" {
		dir = "sup3-data"
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal("invalid SUP3_DATA_DIR")
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		log.Fatal("cannot create SUP3_DATA_DIR")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := sup3.OpenStore(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatal("cannot initialize Sup3API storage")
	}
	engine := sup3.NewEngine(store, dir, sup3.NewProvider("tripo", ""), sup3.NewProvider("meshy", ""))
	engine.ResolveProvider = resolve
	engine.Start(context.Background())
	group := r.Group("/v1/assets", middleware.NewAPIKeyIdentityMiddleware(keys, cfg))
	handle := func(c *gin.Context) {
		key, ok := middleware.GetAPIKeyFromContext(c)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !owners[key.UserID] {
			c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "asset_access_denied", "message": "user is not provisioned for asset provider credits"}})
			return
		}
		if key.GroupID == nil || *key.GroupID <= 0 {
			c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "asset_group_required", "message": "bind the API key to a provider group"}})
			return
		}
		c.Request = c.Request.WithContext(sup3.WithRouting(c.Request.Context(), *key.GroupID))
		if strings.HasPrefix(c.Request.URL.Path, "/providers/") {
			engine.ServeNativeHTTP(c.Writer, c.Request, key.UserID, key.ID)
		} else {
			engine.ServeHTTP(c.Writer, c.Request, key.UserID, key.ID)
		}
	}
	group.GET("/*path", handle)
	group.POST("/*path", handle)
	for _, provider := range []string{"tripo", "meshy"} {
		native := r.Group("/providers/"+provider, middleware.NewAPIKeyIdentityMiddleware(keys, cfg))
		native.GET("/*path", handle)
		native.POST("/*path", handle)
		native.DELETE("/*path", handle)
	}
	log.Print("Sup3API asset routes enabled; billing uses provider-native credits at official rates")
}
