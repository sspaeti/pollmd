package voter

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Salt holds a 32-byte random value that rotates at midnight UTC.
// The salt lives only in memory — rotating it discards the previous day's
// hashing key, which is what makes voter identifiers anonymous-by-construction.
type Salt struct {
	mu      sync.Mutex
	current []byte
	day     string // YYYY-MM-DD UTC of `current`
}

func NewSalt() *Salt {
	s := &Salt{}
	s.mu.Lock()
	s.rotateLocked()
	s.mu.Unlock()
	return s
}

// Current returns today's salt, rotating it if the UTC day has changed since
// the last call.
func (s *Salt) Current() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	today := time.Now().UTC().Format("2006-01-02")
	if today != s.day {
		s.rotateLocked()
	}
	return s.current
}

func (s *Salt) rotateLocked() {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("voter: salt rotation failed: " + err.Error())
	}
	s.current = buf
	s.day = time.Now().UTC().Format("2006-01-02")
}

// Hash returns a 32-character hex digest identifying a voter for one survey.
// Including surveyID in the input means the same reader produces unrelated
// hashes across different newsletters.
func Hash(ip, ua, surveyID string, salt []byte) string {
	h := sha256.New()
	h.Write([]byte(ip))
	h.Write([]byte{0})
	h.Write([]byte(ua))
	h.Write([]byte{0})
	h.Write(salt)
	h.Write([]byte{0})
	h.Write([]byte(surveyID))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:16])
}
