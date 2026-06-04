package server

import "testing"

func TestSlugAccepts(t *testing.T) {
	good := []string{
		"a",
		"2026-06-04",
		"2026-06-04-format",
		"awesome",
		"could_be_better",
		"x9",
	}
	for _, s := range good {
		if !slugRe.MatchString(s) {
			t.Errorf("expected %q to match slug regex", s)
		}
	}
}

func TestSlugRejects(t *testing.T) {
	bad := []string{
		"",
		"-leading",
		"_leading",
		"UpperCase",
		"has space",
		"has/slash",
		"has.dot",
		"semi;colon",
	}
	for _, s := range bad {
		if slugRe.MatchString(s) {
			t.Errorf("expected %q to be rejected", s)
		}
	}
}
