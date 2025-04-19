package slogger

import (
	"context"
	"log"
	"log/slog"
	"testing"
)

/*
=== RUN   TestSlogOutputJson
{
    "time": "2025-04-19T17:21:14.177446+08:00",
    "level": "INFO",
    "source": {
        "function": "github.com/daheige/hephfx/slogger.TestSlogOutputJson",
        "file": "/Users/amanda/web/go/hephfx/slogger/slog_test.go",
        "line": 32
    },
    "msg": "hello",
    "a": 1,
    "name": "go",
    "user": "coco"
}
-- PASS: TestSlogOutputJson (0.00s)
PASS
*/
func TestSlogOutputJson(t *testing.T) {
	// Init(WithEnableJSON())
	Init(WithEnableJSON(), WithAddSource(true))
	slog.Info("hello", slog.String("name", "slog"), slog.Int("number", 10))
	slog.Info("hello", "a", 1, "name", "go")
	slog.Info("hello", "a", 1, "name", "go", slog.String("user", "coco"))
}

/**
=== RUN   TestSlogOutputText
time=2025-04-05T21:55:35.231+08:00 level=INFO msg=hello name=slog number=10
time=2025-04-05T21:55:35.231+08:00 level=INFO msg=hello a=1 name=go
time=2025-04-05T21:55:35.231+08:00 level=INFO msg=hello a=1 name=go user=coco
time=2025-04-05T21:55:35.231+08:00 level=WARN msg="foo waring" a=1 name=go user=coco
--- PASS: TestSlogOutputText (0.00s)
PASS
*/
func TestSlogOutputText(t *testing.T) {
	// output to text
	Init()
	// Init(WithAddSource(true))
	slog.Info("hello", slog.String("name", "slog"), slog.Int("number", 10))
	slog.Info("hello", "a", 1, "name", "go")
	slog.Info("hello", "a", 1, "name", "go", slog.String("user", "coco"))
	slog.Warn("foo waring", "a", 1, "name", "go", slog.String("user", "coco"))
}

func TestSlogOutputFile(t *testing.T) {
	// log to file with json
	Init(
		WithEnableJSON(), WithAddSource(true), WithOutputToFile(true),
		WithFileName("app.log"),
	)

	slog.Debug("debug abc", "a", 1, "b", "234abc") // no log to output

	slog.Info("hello", slog.String("name", "slog"), slog.Int("number", 10))
	slog.Info("hello", "a", 1, "name", "go")
	slog.Info("hello", "a", 1, "name", "go", slog.String("user", "coco"))
	slog.Warn("foo waring", "a", 1, "name", "go", slog.String("user", "coco"))

	// The log standard package will also undergo changes, with the following format
	// {"time":"2025-04-06T10:36:09.51279+08:00","level":"INFO","msg":"abc"}
	log.Println("abc")

	// support custom attributes
	l := slog.Default().With("request_id", "abc")
	l.Info("hello", slog.String("name", "slog"))
	l.Info("world", slog.String("user", "coco"))
}

func TestSlogWith(t *testing.T) {
	Init(WithEnableJSON(), WithAddSource(true))
	// support custom attributes
	l := With(slog.String("request_id", "abc"), "ns", "default")
	l.Info("hello", slog.String("name", "slog"))
	l.Info("world", slog.String("user", "coco"))
}

func TestSlogCtx(t *testing.T) {
	Init(WithEnableJSON(), WithAddSource(true))

	// Using the context package with Slog
	// create context slog handler
	h := ContextHandler{
		Handler: slog.Default().Handler(),
	}
	logger := slog.New(h)
	ctx := AppendCtx(context.Background(), slog.String("request_id", "xxx-123"))
	logger.InfoContext(ctx, "hello", slog.String("uid", "789"))
}

// User user
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LogValue impl LogValuer interface to filter field,
// only output id filed.
// Desensitization of sensitive data.
func (u *User) LogValue() slog.Value {
	return slog.StringValue(u.ID)
}

func TestSlogLogValue(t *testing.T) {
	u := &User{
		ID:       "abc",
		Email:    "abc@example.com",
		Password: "abc",
	}

	Init(WithEnableJSON(), WithAddSource(true))
	slog.Info("hello", slog.Int("number", 10), "user", u)
}
