package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/clickTwice26/fatman/services/api/internal/service"
)

const authClaimsKey ctxKey = "auth_claims"

// AccessTokenDenylist checks whether a JWT's JTI has been revoked in Redis.
type AccessTokenDenylist interface {
	IsAccessTokenDenied(ctx context.Context, jti string) (bool, error)
}

// AuthClaimsFromContext returns authenticated JWT claims attached by AuthMiddleware.
func AuthClaimsFromContext(ctx context.Context) (service.AccessTokenClaims, bool) {
	claims, ok := ctx.Value(authClaimsKey).(service.AccessTokenClaims)
	return claims, ok
}

// AuthMiddleware verifies the Bearer JWT and ensures its JTI is not in the Redis denylist.
func AuthMiddleware(tokens *service.TokenService, denylist AccessTokenDenylist) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			scheme, rawToken, ok := strings.Cut(authHeader, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(rawToken) == "" {
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization header", nil)
				return
			}

			claims, err := tokens.ParseAccessToken(strings.TrimSpace(rawToken))
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token", nil)
				return
			}

			if denylist != nil {
				denied, err := denylist.IsAccessTokenDenied(r.Context(), claims.JTI)
				if err != nil || denied {
					WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "access token has been revoked", nil)
					return
				}
			}

			ctx := context.WithValue(r.Context(), authClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
