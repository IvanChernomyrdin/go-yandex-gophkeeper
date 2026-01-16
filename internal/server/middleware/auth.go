// Проверка JWT / access токенов
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

type JWTVerifier struct {
	SigningKey string
	Issuer     string
	Audience   string
}

func NewJWTVerifier(signingKey, issuer, audience string) *JWTVerifier {
	return &JWTVerifier{SigningKey: signingKey, Issuer: issuer, Audience: audience}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(userIDKey)
	s, ok := v.(string)
	return s, ok
}

func (v *JWTVerifier) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearer(r.Header.Get("Authorization"))
			if tokenStr == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			claims := &jwt.RegisteredClaims{}

			parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
			_, err := parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				return []byte(v.SigningKey), nil
			})

			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					http.Error(w, "token expired", http.StatusUnauthorized)
					return
				}
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if v.Issuer != "" && claims.Issuer != v.Issuer {
				http.Error(w, "invalid token issuer", http.StatusUnauthorized)
				return
			}

			if v.Audience != "" {
				ok := false
				for _, aud := range claims.Audience {
					if aud == v.Audience {
						ok = true
						break
					}
				}
				if !ok {
					http.Error(w, "invalid token audience", http.StatusUnauthorized)
					return
				}
			}

			userID := strings.TrimSpace(claims.Subject)
			if userID == "" {
				http.Error(w, "invalid token subject", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
