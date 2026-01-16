package tests

import (
	"bytes"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
)

func TestNewVersionCmd_PrintsVersionAndBuildDate(t *testing.T) {
	const (
		version   = "1.2.3"
		buildDate = "2026-01-16"
	)

	cmd := cli.NewVersionCmd(version, buildDate)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// у команды нет аргументов
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "version=1.2.3") {
		t.Fatalf("expected version output, got %q", got)
	}
	if !strings.Contains(got, "build_date=2026-01-16") {
		t.Fatalf("expected build_date output, got %q", got)
	}
}
