package token

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	errs "github.com/pkg/errors"
)

// ReadManagerFromContext extracts the token manager from the context
func ReadManagerFromContext(ctx context.Context) (*Manager, error) {
	tm := tokencontext.ReadTokenManagerFromContext(ctx)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")

		return nil, errs.New("Missing token manager")
	}
	tokenManager := tm.(Manager)
	return &tokenManager, nil
}
