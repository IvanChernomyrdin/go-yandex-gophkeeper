package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
)

func TestNewRootCmd_HasExpectedSubcommands(t *testing.T) {
	cmd := cli.NewRootCmd("1.0.0", "2026-01-16")

	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}

	want := []string{"register", "login", "refresh", "version"}
	for _, w := range want {
		if !names[w] {
			t.Fatalf("expected subcommand %q to exist", w)
		}
	}
}

func TestNewRootCmd_PersistentPreRunE_LoadsCreds(t *testing.T) {
	// Подготовим дефолтный путь и валидный файл кредов ДО запуска команды.
	p, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}

	// подчистим после теста
	t.Cleanup(func() { _ = os.Remove(p) })

	// убедимся, что директория существует (если Save не делает mkdir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	wantCreds := &config.Credentials{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
	}
	if err := config.Save(p, wantCreds); err != nil {
		t.Fatalf("Save creds: %v", err)
	}

	root := cli.NewRootCmd("1.0.0", "2026-01-16")

	// Важно: чтобы выполнить PersistentPreRunE, нужно реально запустить команду.
	// Возьмём безопасную подкоманду version, она не ходит в сеть/файлы кроме PreRun.
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Не можем напрямую достать app из NewRootCmd (он внутри замыкания),
	// но можем проверить косвенно: команда должна успешно выполниться,
	// а вывод version должен быть.
	got := out.String()
	if !strings.Contains(got, "version=") || !strings.Contains(got, "build_date=") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNewRootCmd_PersistentPreRunE_ReturnsErrorOnBadCredsFile(t *testing.T) {
	p, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(p) })

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Создадим битый файл (невалидный формат для config.Load)
	if err := os.WriteFile(p, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := cli.NewRootCmd("1.0.0", "2026-01-16")
	root.SetArgs([]string{"version"})

	err = root.Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
