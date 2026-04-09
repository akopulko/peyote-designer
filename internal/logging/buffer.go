package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Line    string
}

type Buffer struct {
	mu          sync.RWMutex
	capacity    int
	entries     []Entry
	subscribers []func()
}

func NewBuffer(capacity int) *Buffer {
	return &Buffer{
		capacity: capacity,
		entries:  make([]Entry, 0, capacity),
	}
}

func (b *Buffer) Append(entry Entry) {
	b.mu.Lock()
	if len(b.entries) == b.capacity {
		b.entries = append(b.entries[:0], b.entries[1:]...)
	}
	b.entries = append(b.entries, entry)
	subscribers := append([]func(){}, b.subscribers...)
	b.mu.Unlock()

	for _, subscriber := range subscribers {
		subscriber()
	}
}

func (b *Buffer) Entries() []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]Entry, len(b.entries))
	copy(out, b.entries)
	return out
}

func (b *Buffer) Subscribe(fn func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, fn)
}

func (b *Buffer) Clear() {
	b.mu.Lock()
	b.entries = b.entries[:0]
	subscribers := append([]func(){}, b.subscribers...)
	b.mu.Unlock()

	for _, subscriber := range subscribers {
		subscriber()
	}
}

type BufferHandler struct {
	buffer *Buffer
	attrs  []slog.Attr
	group  string
	level  slog.Level
}

func NewBufferHandler(buffer *Buffer) *BufferHandler {
	return &BufferHandler{
		buffer: buffer,
		level:  slog.LevelDebug,
	}
}

func (h *BufferHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *BufferHandler) Handle(_ context.Context, record slog.Record) error {
	parts := []string{record.Time.Format(time.RFC3339), record.Level.String(), record.Message}
	record.Attrs(func(attr slog.Attr) bool {
		parts = append(parts, fmt.Sprintf("%s=%v", attr.Key, attr.Value.Any()))
		return true
	})
	for _, attr := range h.attrs {
		parts = append(parts, fmt.Sprintf("%s=%v", attr.Key, attr.Value.Any()))
	}
	line := strings.Join(parts, " ")
	h.buffer.Append(Entry{
		Time:    record.Time,
		Level:   record.Level,
		Message: record.Message,
		Line:    line,
	})
	return nil
}

func (h *BufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := append([]slog.Attr{}, h.attrs...)
	combined = append(combined, attrs...)
	return &BufferHandler{buffer: h.buffer, attrs: combined, group: h.group, level: h.level}
}

func (h *BufferHandler) WithGroup(name string) slog.Handler {
	group := name
	if h.group != "" {
		group = h.group + "." + name
	}
	return &BufferHandler{buffer: h.buffer, attrs: h.attrs, group: group, level: h.level}
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithAttrs(attrs))
	}
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithGroup(name))
	}
	return &MultiHandler{handlers: handlers}
}

func NewLogger(buffer *Buffer) *slog.Logger {
	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(NewMultiHandler(textHandler, NewBufferHandler(buffer)))
}
