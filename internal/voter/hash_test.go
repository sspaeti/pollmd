package voter

import "testing"

func TestHashIsDeterministic(t *testing.T) {
	salt := []byte("test-salt-test-salt-test-salt-aa")
	a := Hash("1.2.3.4", "Mozilla/5.0", "2026-06-04", salt)
	b := Hash("1.2.3.4", "Mozilla/5.0", "2026-06-04", salt)
	if a != b {
		t.Fatalf("expected same hash for same inputs, got %q vs %q", a, b)
	}
}

func TestHashSeparatesSurveys(t *testing.T) {
	salt := []byte("test-salt-test-salt-test-salt-aa")
	a := Hash("1.2.3.4", "Mozilla/5.0", "issue-a", salt)
	b := Hash("1.2.3.4", "Mozilla/5.0", "issue-b", salt)
	if a == b {
		t.Fatalf("hash must differ across surveys to prevent cross-issue linking")
	}
}

func TestHashSeparatesReaders(t *testing.T) {
	salt := []byte("test-salt-test-salt-test-salt-aa")
	a := Hash("1.2.3.4", "Mozilla/5.0", "issue-a", salt)
	b := Hash("9.9.9.9", "Mozilla/5.0", "issue-a", salt)
	if a == b {
		t.Fatalf("different IPs should produce different hashes")
	}
}

func TestHashSeparatesByUA(t *testing.T) {
	salt := []byte("test-salt-test-salt-test-salt-aa")
	a := Hash("1.2.3.4", "Mozilla/5.0", "issue-a", salt)
	b := Hash("1.2.3.4", "Curl/8", "issue-a", salt)
	if a == b {
		t.Fatalf("different UAs should produce different hashes")
	}
}

func TestHashLengthIs32HexChars(t *testing.T) {
	salt := []byte("test-salt-test-salt-test-salt-aa")
	h := Hash("1.2.3.4", "Mozilla/5.0", "issue-a", salt)
	if len(h) != 32 {
		t.Fatalf("expected 32 hex chars, got %d (%q)", len(h), h)
	}
}

func TestSaltRotation(t *testing.T) {
	s := NewSalt()
	first := append([]byte(nil), s.Current()...)
	// Force-expire the cached day so the next Current() call rotates.
	s.mu.Lock()
	s.day = ""
	s.mu.Unlock()
	second := s.Current()
	if string(first) == string(second) {
		t.Fatalf("salt did not rotate")
	}
	if len(second) != 32 {
		t.Fatalf("rotated salt should be 32 bytes, got %d", len(second))
	}
}

func TestSaltIsStableWithinDay(t *testing.T) {
	s := NewSalt()
	a := append([]byte(nil), s.Current()...)
	b := s.Current()
	if string(a) != string(b) {
		t.Fatalf("salt changed within the same UTC day")
	}
}
