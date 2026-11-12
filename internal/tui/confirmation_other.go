//go:build !windows

package tui

import (
	"context"
	"github.com/muesli/cancelreader"
	"io"
	"os"
	"ssh-keyflow/internal/app"
)

func readNativeConfirmation(ctx context.Context, input io.Reader) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	// Cancel only this exact approval read; native processes retain original stdin.
	if _, ok := input.(*os.File); ok {
		reader, err := cancelreader.NewReader(input)
		if err != nil {
			return false, err
		}
		defer reader.Close()
		stopped := make(chan struct{})
		stop := context.AfterFunc(ctx, func() { reader.Cancel(); close(stopped) })
		defer func() {
			if !stop() {
				<-stopped
			}
		}()
		input = reader
	}
	yes, err := app.ReadConfirmation(input)
	if interrupted := ctx.Err(); interrupted != nil {
		return false, interrupted
	}
	return yes, err
}

// cancellableInput lets Ctrl+C stop a typed paste. It is used only for
// operations that start no native process, so no process shares this stdin.
func cancellableInput(ctx context.Context, input io.Reader) (io.Reader, func(), error) {
	if _, ok := input.(*os.File); !ok {
		return input, func() {}, nil
	}
	reader, err := cancelreader.NewReader(input)
	if err != nil {
		return nil, nil, err
	}
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { reader.Cancel(); close(stopped) })
	return reader, func() {
		if !stop() {
			<-stopped
		}
		reader.Close()
	}, nil
}
