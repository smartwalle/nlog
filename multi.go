package nlog

import (
	"context"
	"log/slog"
)

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	var h = &MultiHandler{}
	h.handlers = handlers
	return h
}

func (this *MultiHandler) Add(handler slog.Handler) *MultiHandler {
	if handler != nil {
		this.handlers = append(this.handlers, handler)
	}
	return this
}

func (this *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for i := range this.handlers {
		if this.handlers[i].Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (this *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for i := range this.handlers {
		if this.handlers[i].Enabled(ctx, r.Level) {
			if err := this.handlers[i].Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var nHandlers = make([]slog.Handler, len(this.handlers))
	for i := range this.handlers {
		nHandlers[i] = this.handlers[i].WithAttrs(attrs)
	}
	return &MultiHandler{handlers: nHandlers}
}

func (this *MultiHandler) WithGroup(name string) slog.Handler {
	var nHandlers = make([]slog.Handler, len(this.handlers))
	for i := range this.handlers {
		nHandlers[i] = this.handlers[i].WithGroup(name)
	}
	return &MultiHandler{handlers: nHandlers}
}
