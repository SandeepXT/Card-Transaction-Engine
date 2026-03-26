package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/SandeepXT/Card-Transaction-Engine/internal/models"
)

type MemoryStore struct {
	mu      sync.RWMutex
	cards   map[string]*models.Card
	history map[string][]*models.Transaction
	seq     int64
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		cards:   make(map[string]*models.Card),
		history: make(map[string][]*models.Transaction),
	}
	s.seed()
	return s
}

func (s *MemoryStore) seed() {
	s.cards["4123456789012345"] = &models.Card{
		CardNumber: "4123456789012345",
		CardHolder: "John Doe",
		PinHash:    "03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4",
		Balance:    1000.00,
		Status:     models.Active,
	}
	s.cards["4987654321098765"] = &models.Card{
		CardNumber: "4987654321098765",
		CardHolder: "Jane Smith",
		PinHash:    "f8638b979b2f4f793ddb6dbd197e0ee25a7a6ea32b0ae22f5e3c5d119d839e75",
		Balance:    2500.00,
		Status:     models.Active,
	}
	s.cards["4111111111111111"] = &models.Card{
		CardNumber: "4111111111111111",
		CardHolder: "Bob Wilson",
		PinHash:    "888df25ae35772424a560c7152a1de794440e0ea5cfee62828333a456a506e05",
		Balance:    500.00,
		Status:     models.Blocked,
	}
}

func (s *MemoryStore) FindCard(number string) *models.Card {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cards[number]
}

// Fix #4: Withdraw returns (newBalance, success) atomically.
// Both the sufficiency check and the deduction happen inside a single
// exclusive lock, eliminating any TOCTOU race under concurrent requests.
func (s *MemoryStore) Withdraw(number string, amount float64) (float64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[number]
	if !ok || c.Balance < amount {
		return 0, false
	}
	c.Balance -= amount
	return c.Balance, true
}

// Fix #4: Topup returns the new balance atomically.
func (s *MemoryStore) Topup(number string, amount float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.cards[number]; ok {
		c.Balance += amount
		return c.Balance
	}
	return 0
}

func (s *MemoryStore) Record(txn *models.Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[txn.CardNumber] = append(s.history[txn.CardNumber], txn)
}

func (s *MemoryStore) History(number string) []*models.Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.history[number]
	out := make([]*models.Transaction, len(src))
	copy(out, src)
	return out
}

func (s *MemoryStore) NextID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("TXN%d%04d", time.Now().UnixNano(), s.seq)
}

func (s *MemoryStore) Cards() []*models.Card {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Card, 0, len(s.cards))
	for _, c := range s.cards {
		out = append(out, c)
	}
	return out
}