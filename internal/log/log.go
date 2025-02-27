package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
)

const originalKeyError = "error"

func AttrError(err error) slog.Attr {
	return slog.Attr{Key: originalKeyError, Value: slog.AnyValue(err)}
}

func Setup() {
	slog.SetDefault(NewLogger(os.Stderr))
}

func NewLogger(out io.Writer) *slog.Logger {
	h := slog.NewJSONHandler(out, &slog.HandlerOptions{})
	return slog.New(h)
}

func NewErrorAttributeTransformer(h slog.Handler) slog.Handler {
	return &ErrorAttributeTransformer{h}
}

type ErrorAttributeTransformer struct {
	slog.Handler
}

var _ slog.Handler = (*ErrorAttributeTransformer)(nil)

func (h *ErrorAttributeTransformer) Handle(ctx context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, record.NumAttrs())
	idx := -1
	foundIndex := -1
	record.Attrs(func(a slog.Attr) bool {
		idx++
		attrs[idx] = a
		if a.Key == originalKeyError && a.Value.Kind() == slog.KindAny {
			foundIndex = idx
		}
		return false
	})
	if foundIndex < 0 {
		return h.Handler.Handle(ctx, record)
	}

	orig := attrs[foundIndex]
	valErr, ok := orig.Value.Any().(error)
	if !ok {
		return h.Handler.Handle(ctx, record)
	}

	newAttrs := slices.Clone(attrs)
	newAttrs = slices.Delete(newAttrs, foundIndex, foundIndex+1)
	newAttrs = slices.Grow(newAttrs, 1)
	newAttrs = slices.Insert(newAttrs, foundIndex, slog.String("error.message", valErr.Error()), slog.String("error.type", fmt.Sprintf("%T", valErr)))
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	newRecord.AddAttrs(newAttrs...)
	return h.Handler.Handle(ctx, newRecord)
}
