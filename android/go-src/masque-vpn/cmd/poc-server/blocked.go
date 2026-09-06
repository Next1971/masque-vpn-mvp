package main

import (
	"os"
	"strings"
)

const defaultBlockedFile = "/opt/masque/blocked_cns"

func loadBlockedSet(fromTOML []string, extraFile string) map[string]struct{} {
	m := make(map[string]struct{})
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") {
			return
		}
		m[s] = struct{}{}
	}
	for _, s := range fromTOML {
		add(s)
	}
	if extraFile == "" {
		return m
	}
	raw, err := os.ReadFile(extraFile)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(raw), "\n") {
		add(line)
	}
	return m
}
