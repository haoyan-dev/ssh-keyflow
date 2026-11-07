package tui

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	tea "charm.land/bubbletea/v2"
)

// Own OS interrupts so an Exec restoration race cannot kill the returned TUI.
// Caller WithFilter options still receive the ordinary Bubble Tea messages.
type handoffSignals struct {
	sync.Mutex
	program *tea.Program
	cancel  context.CancelFunc
}

func (s *handoffSignals) start(cancel context.CancelFunc) {
	s.Lock()
	defer s.Unlock()
	s.cancel = cancel
}
func (s *handoffSignals) finish() {
	s.Lock()
	defer s.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}
func (s *handoffSignals) listen() tea.Msg {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	done := make(chan struct{})
	go func() { s.program.Wait(); close(done) }()
	defer s.finish()
	for {
		select {
		case <-done:
			return nil
		case sig := <-signals:
			s.Lock()
			cancel := s.cancel
			s.Unlock()
			if sig == syscall.SIGINT && cancel != nil {
				cancel()
				continue
			}
			if sig == syscall.SIGINT {
				s.program.Send(tea.InterruptMsg{})
			} else {
				s.program.Send(tea.QuitMsg{})
			}
		}
	}
}
