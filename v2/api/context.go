package api

import "context"

type ContextKey int

const (
	ContextKeyJWTToken ContextKey = iota

	ContextKeyAuthClaims

	ContextKeyAPIKey
)

func GetAPIKeyFromContext(ctx context.Context) string {
	apiKey, ok := ctx.Value(ContextKeyAPIKey).(string)
	if !ok {
		return ""
	}

	return apiKey
}
