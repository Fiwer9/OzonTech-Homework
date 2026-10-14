package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"awesomeProject/pkg/logger"
)

func TestLoggerInfoAndError(t *testing.T) {
	var out, errBuf bytes.Buffer
	log := logger.NewWithWriters(&out, &errBuf)
	log.Info("hello %s", "world")
	log.Error("fail %d", 1)

	if !strings.Contains(out.String(), "hello world") {
		t.Fatalf("info: %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "fail 1") {
		t.Fatalf("error: %q", errBuf.String())
	}
}
