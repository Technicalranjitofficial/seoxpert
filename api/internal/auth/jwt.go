package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// SupabaseClaims mirrors the JWT payload that Supabase GoTrue issues.
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email    string                 `json:"email"`
	Phone    string                 `json:"phone"`
	Role     string                 `json:"role"`
	AppMeta  map[string]interface{} `json:"app_metadata"`
	UserMeta map[string]interface{} `json:"user_metadata"`
}

// ParseToken validates a Supabase JWT against the JWT secret (HS256).
// Returns the claims so middleware can extract user ID and role.
func ParseToken(tokenStr, secret string) (*SupabaseClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &SupabaseClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithAudience("authenticated"))

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*SupabaseClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}
