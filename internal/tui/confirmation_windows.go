package tui

import (
	"context"
	"io"
	"ssh-keyflow/internal/app"
)

// Console Ctrl+C aborts the original ReadFile; never wrap SSH credential stdin.
func readNativeConfirmation(ctx context.Context, input io.Reader) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	yes, err := app.ReadConfirmation(input)
	if interrupted := ctx.Err(); interrupted != nil {
		return false, interrupted
	}
	return yes, err
}
