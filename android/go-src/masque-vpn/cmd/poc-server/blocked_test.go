package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBlockedSet(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "blocked_cns")
	if err := os.WriteFile(p, []byte("masque-client-7\n# comment\n\nmasque-client-7\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m := loadBlockedSet([]string{" masque-client-3 ", ""}, p)
	if _, ok := m["masque-client-7"]; !ok {
		t.Fatal("file CN")
	}
	if _, ok := m["masque-client-3"]; !ok {
		t.Fatal("toml CN")
	}
	if _, ok := m["# comment"]; ok {
		t.Fatal("comment")
	}
	if n := len(m); n != 2 {
		t.Fatalf("len %d", n)
	}
}
