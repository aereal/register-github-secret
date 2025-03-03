package log_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"testing"
	"testing/slogtest"

	"github.com/aereal/register-github-secret/internal/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestHandler_error_replace(t *testing.T) {
	testCases := []struct {
		runLog func(l *slog.Logger)
		want   func() []map[string]any
		name   string
	}{
		{
			name:   "with no error attribute",
			runLog: func(l *slog.Logger) { l.Info("msg", slog.String("s", "a")) },
			want: func() []map[string]any {
				return []map[string]any{
					{
						slog.LevelKey:   "INFO",
						slog.MessageKey: "msg",
						"s":             "a",
					},
				}
			},
		},
		{
			name:   "with simple error",
			runLog: func(l *slog.Logger) { l.Info("msg", slog.Any("error", errors.New("oops"))) },
			want: func() []map[string]any {
				return []map[string]any{
					{
						slog.LevelKey:   "INFO",
						slog.MessageKey: "msg",
						"error.message": "oops",
						"error.type":    "*errors.errorString",
					},
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			h := log.NewErrorAttributeTransformer(slog.NewJSONHandler(out, &slog.HandlerOptions{}))
			logger := slog.New(h)
			tc.runLog(logger)
			got, err := collectLogEntries(out)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.want()
			if diff := cmp.Diff(want, got, ignoreTimeAttribute()); diff != "" {
				t.Errorf("result (-want, +got):\n%s", diff)
			}
		})
	}
}

func ignoreTimeAttribute() cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ any) bool {
		return key == slog.TimeKey
	})
}

func collectLogEntries(b *bytes.Buffer) ([]map[string]any, error) {
	rets := []map[string]any{}
	for {
		line, err := b.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		m := map[string]any{}
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, err
		}
		rets = append(rets, m)
	}
	return rets, nil
}

func TestHandler(t *testing.T) {
	out := new(bytes.Buffer)
	newHandler := func(_ *testing.T) slog.Handler {
		out.Reset()
		jh := slog.NewJSONHandler(out, &slog.HandlerOptions{})
		return log.NewErrorAttributeTransformer(jh)
	}
	newResult := func(_ *testing.T) map[string]any {
		ret := map[string]any{}
		for {
			line, err := out.ReadBytes('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			m := map[string]any{}
			if err := json.Unmarshal(line, &m); err != nil {
				t.Fatal(err)
			}
			maps.Insert(ret, maps.All(m))
		}
		return ret
	}
	slogtest.Run(t, newHandler, newResult)
}
