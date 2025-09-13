package apperr

import (
	"context"

	"github.com/m-mizutani/ctxlog"
)

func Handle(ctx context.Context, err error) {
	logger := ctxlog.From(ctx)
	logger.Error("application error", "error", err)
}
