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

func (h *MultiHandler) Add(handler slog.Handler) *MultiHandler {
	if handler != nil {
		h.handlers = append(h.handlers, handler)
	}
	return h
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for i := range h.handlers {
		if h.handlers[i].Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for i := range h.handlers {
		if h.handlers[i].Enabled(ctx, r.Level) {
			if err := h.handlers[i].Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var nHandlers = make([]slog.Handler, len(h.handlers))
	for i := range h.handlers {
		nHandlers[i] = h.handlers[i].WithAttrs(attrs)
	}
	return &MultiHandler{handlers: nHandlers}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	var nHandlers = make([]slog.Handler, len(h.handlers))
	for i := range h.handlers {
		nHandlers[i] = h.handlers[i].WithGroup(name)
	}
	return &MultiHandler{handlers: nHandlers}
}
