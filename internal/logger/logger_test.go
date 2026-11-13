package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fleettrack/internal/requestid"
	"strings"
	"testing"
)

// TestStdLogger_RequestFields проверяет, что запись с контекстом запроса несёт request_id
// и поля, дописанные позже по цепочке middleware
func TestStdLogger_RequestFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, InfoLevel, FormatJSON)

	ctx := requestid.NewContext(context.Background(), "req-1")
	ctx = WithRequestFields(ctx)
	AddRequestFields(ctx, "user_id", 42, "org_id", 7)

	log.Log(ctx, InfoLevel, "request", "status", 200)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("not a JSON line: %q", buf.String())
	}
	want := map[string]any{"msg": "request", "request_id": "req-1", "user_id": 42.0, "org_id": 7.0, "status": 200.0}
	for k, v := range want {
		if rec[k] != v {
			t.Errorf("%s = %v, want %v", k, rec[k], v)
		}
	}
}

// TestStdLogger_Level проверяет, что записи ниже уровня не пишутся
func TestStdLogger_Level(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WarnLevel, FormatText)

	log.Info("hidden")
	log.Warn("shown")

	out := buf.String()
	if strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Errorf("output = %q", out)
	}
}

// TestAddRequestFields_NoContainer проверяет, что без контейнера в контексте поля просто не пишутся
func TestAddRequestFields_NoContainer(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, InfoLevel, FormatJSON)

	ctx := context.Background()
	AddRequestFields(ctx, "user_id", 1)
	log.Log(ctx, InfoLevel, "no fields")

	if strings.Contains(buf.String(), "user_id") || strings.Contains(buf.String(), "request_id") {
		t.Errorf("unexpected fields: %q", buf.String())
	}
}

func TestParseFormat(t *testing.T) {
	for in, want := range map[string]Format{"": FormatJSON, "JSON": FormatJSON, "text": FormatText} {
		got, err := ParseFormat(in)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(xml) должен вернуть ошибку")
	}
}
