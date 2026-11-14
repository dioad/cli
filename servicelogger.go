package cli

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ServiceLogger returns a context whose logger carries service=name. It enriches
// the logger already in ctx, or the process logger if ctx has none, and it does
// not touch any zerolog global, so several services in one process can each
// name themselves without racing over shared state.
//
// A shutdown path should derive its context with context.WithoutCancel(ctx),
// not context.Background(), so it keeps the service label.
func ServiceLogger(ctx context.Context, name string) context.Context {
	base := *zerolog.Ctx(ctx)
	if base.GetLevel() == zerolog.Disabled {
		// ctx carries no logger and no default is set.
		base = log.Logger
	}

	return base.With().Str("service", name).Logger().WithContext(ctx)
}
