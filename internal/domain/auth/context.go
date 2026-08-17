package auth

import "context"

type contextKey string

const claimsKey contextKey = "auth_claims"

func WithContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func FromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

func UserIDFromContext(ctx context.Context) string {
	claims := FromContext(ctx)
	if claims == nil {
		return ""
	}
	return claims.UserID
}
