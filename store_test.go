package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestStoreRoundtrip(t *testing.T) {
	s := NewStore()

	f := Flag{Key: "feature-x", Enabled: true, Description: "desc", RolloutPercent: 50}
	if err := s.Create(f); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Create(f); err == nil {
		t.Fatal("expected error when creating an existing key")
	}

	got, ok := s.Get("feature-x")
	if !ok {
		t.Fatal("expected flag to exist after Create")
	}
	if got != f {
		t.Fatalf("Get returned %+v, want %+v", got, f)
	}

	if _, ok := s.Get("missing"); ok {
		t.Fatal("expected missing key to report not found")
	}

	// Update only Enabled; optional fields must be preserved.
	upd := flagUpdate{Enabled: false}
	updated, err := s.Update("feature-x", upd)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Enabled {
		t.Fatal("expected Enabled to be false after update")
	}
	if updated.Description != "desc" {
		t.Fatalf("expected Description to be preserved, got %q", updated.Description)
	}
	if updated.RolloutPercent != 50 {
		t.Fatalf("expected RolloutPercent to be preserved, got %d", updated.RolloutPercent)
	}

	if _, err := s.Update("missing", upd); err == nil {
		t.Fatal("expected error when updating an unknown key")
	}

	if err := s.Create(Flag{Key: "aaa", Enabled: true}); err != nil {
		t.Fatalf("Create aaa: %v", err)
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(list))
	}
	if list[0].Key != "aaa" || list[1].Key != "feature-x" {
		t.Fatalf("expected keys sorted [aaa feature-x], got %v", list)
	}

	if !s.Delete("aaa") {
		t.Fatal("expected Delete to succeed for an existing key")
	}
	if s.Delete("aaa") {
		t.Fatal("expected second Delete to report false")
	}
	if len(s.List()) != 1 {
		t.Fatalf("expected 1 flag after delete, got %d", len(s.List()))
	}
}

func TestStoreConcurrency(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("flag-%d", n%10)
			_ = s.Create(Flag{Key: key, Enabled: true, RolloutPercent: n % 101})
			_, _ = s.Get(key)
			_ = s.List()
		}(i)
	}
	wg.Wait()

	if got := len(s.List()); got != 10 {
		t.Fatalf("expected 10 distinct flags, got %d", got)
	}
}
