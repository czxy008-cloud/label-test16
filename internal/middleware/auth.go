package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"filestore/internal/config"
)

// TokenAuth 基于请求头 Token 的鉴权中间件
// 从配置中读取合法 token 列表，在请求头中查找配置指定的 token 头字段。
func TokenAuth(cfg config.AuthConfig) gin.HandlerFunc {
	header := cfg.TokenHeader
	if header == "" {
		header = "X-Auth-Token"
	}
	validSet := make(map[string]struct{}, len(cfg.ValidTokens))
	for _, t := range cfg.ValidTokens {
		validSet[strings.TrimSpace(t)] = struct{}{}
	}

	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader(header))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "missing auth token",
			})
			return
		}
		if _, ok := validSet[token]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "invalid auth token",
			})
			return
		}
		c.Next()
	}
}
