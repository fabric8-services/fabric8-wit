// Package token_context contains the code that extract token manager from the
// context.
package token_context

import (
	"golang.org/x/net/context"
)

type contextTMKey int

const (
	//contextTokenManagerKey is a key that will be used to put and to get `tokenManager` from goa.context
	contextTokenManagerKey contextTMKey = iota + 1
)

// ReadTokenManagerFromContext returns an interface that encapsulates the
// tokenManager extracted from context. This interface can be safely converted.
// Must have been set by ContextWithTokenManager ONLY.
func ReadTokenManagerFromContext(ctx context.Context) interface{} {
	tm := ctx.Value(contextTokenManagerKey)
	if tm != nil {
		return tm
	}
	return nil
}

// ContextWithTokenManager injects tokenManager in the context for every incoming request
// Accepts Token.Manager in order to make sure that correct object is set in the context.
// Only other possible value is nil
func ContextWithTokenManager(ctx context.Context, tm interface{}) context.Context {
	return context.WithValue(ctx, contextTokenManagerKey, tm)
}
