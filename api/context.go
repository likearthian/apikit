package api

import "context"

type ContextKey int

const (
	ContextKeyJWTToken ContextKey = iota

	ContextKeyAuthClaims

	ContextKeyApikey
)

func GetApikeyFromContext(ctx context.Context) string {
	apikey, ok := ctx.Value(ContextKeyApikey).(string)
	if !ok {
		return ""
	}

	return apikey
}
