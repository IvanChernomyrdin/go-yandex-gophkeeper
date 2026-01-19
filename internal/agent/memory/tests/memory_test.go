package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

func TestNewSecrets_Empty(t *testing.T) {
	s := memory.NewSecrets()
	if s == nil {
		t.Fatalf("expected non-nil store")
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("expected empty list, got %d", len(got))
	}
}

func TestSecretsStore_Get_NotFound(t *testing.T) {
	s := memory.NewSecrets()
	_, err := s.Get("missing")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, serr.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestSecretsStore_ReplaceAll_AndGet(t *testing.T) {
	s := memory.NewSecrets()
	now := time.Now()

	sec := memory.Secret{
		ID:        "id1",
		Type:      "text",
		Title:     "t",
		Payload:   "p",
		Version:   1,
		UpdatedAt: now,
		CreatedAt: now,
	}

	s.ReplaceAll([]memory.Secret{sec})

	got, err := s.Get("id1")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.ID != "id1" || got.Type != "text" || got.Payload != "p" || got.Version != 1 {
		t.Fatalf("unexpected secret: %+v", got)
	}
}

func TestSecretsStore_List_ReturnsAll(t *testing.T) {
	s := memory.NewSecrets()
	now := time.Now()

	s.ReplaceAll([]memory.Secret{
		{ID: "a", Type: "text", Title: "A", Payload: "PA", Version: 1, UpdatedAt: now, CreatedAt: now},
		{ID: "b", Type: "text", Title: "B", Payload: "PB", Version: 2, UpdatedAt: now, CreatedAt: now},
	})

	items := s.List()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// проверяем, что оба ID присутствуют
	seen := map[string]bool{}
	for _, it := range items {
		seen[it.ID] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("expected ids a and b, got %+v", seen)
	}
}

func TestSecretsStore_UpdateFromDB_UpdatesOnlyProvidedFields(t *testing.T) {
	s := memory.NewSecrets()
	now := time.Now()

	metaOld := `{"k":"old"}`
	s.ReplaceAll([]memory.Secret{
		{
			ID:        "id1",
			Type:      "text",
			Title:     "title1",
			Payload:   "payload1",
			Meta:      &metaOld,
			Version:   1,
			UpdatedAt: now.Add(-time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
	})

	// меняем только Type и Payload, meta не трогаем (nil)
	newType := "binary"
	newPayload := "payload2"

	before, _ := s.Get("id1")

	if err := s.UpdateFromDB("id1", &newType, &newPayload, nil); err != nil {
		t.Fatalf("UpdateFromDB error: %v", err)
	}

	after, _ := s.Get("id1")

	if after.Type != "binary" {
		t.Fatalf("expected type=binary, got %q", after.Type)
	}
	if after.Payload != "payload2" {
		t.Fatalf("expected payload=payload2, got %q", after.Payload)
	}
	if after.Meta == nil || *after.Meta != metaOld {
		t.Fatalf("expected meta unchanged, got %+v", after.Meta)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("expected UpdatedAt to be updated, before=%v after=%v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestSecretsStore_UpdateFromDB_UpdatesMeta(t *testing.T) {
	s := memory.NewSecrets()
	now := time.Now()

	s.ReplaceAll([]memory.Secret{
		{ID: "id1", Type: "text", Title: "t", Payload: "p", Version: 1, UpdatedAt: now, CreatedAt: now},
	})

	meta := `{"env":"test"}`
	if err := s.UpdateFromDB("id1", nil, nil, &meta); err != nil {
		t.Fatalf("UpdateFromDB error: %v", err)
	}

	got, _ := s.Get("id1")
	if got.Meta == nil || *got.Meta != meta {
		t.Fatalf("expected meta=%q, got %+v", meta, got.Meta)
	}
}

func TestSecretsStore_UpdateFromDB_NotFound(t *testing.T) {
	s := memory.NewSecrets()
	newType := "text"

	err := s.UpdateFromDB("missing", &newType, nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, serr.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestSecretsStore_Delete_Success(t *testing.T) {
	s := memory.NewSecrets()
	now := time.Now()

	s.ReplaceAll([]memory.Secret{
		{ID: "id1", Type: "text", Title: "t", Payload: "p", Version: 1, UpdatedAt: now, CreatedAt: now},
	})

	if err := s.Delete("id1"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	_, err := s.Get("id1")
	if err == nil {
		t.Fatalf("expected not found after delete")
	}
	if !errors.Is(err, serr.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestSecretsStore_Delete_NotFound(t *testing.T) {
	s := memory.NewSecrets()
	err := s.Delete("missing")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, serr.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}
