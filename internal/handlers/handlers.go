package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/SandeepXT/Card-Transaction-Engine/internal/models"
	"github.com/SandeepXT/Card-Transaction-Engine/internal/store"
)

func Transaction(db *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			reject(w, http.StatusMethodNotAllowed, "05", "method not allowed")
			return
		}

		var req models.TransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			reject(w, http.StatusBadRequest, "05", "invalid request body")
			return
		}

		// Fix #3: validate cardNumber is exactly 16 numeric digits
		if len(req.CardNumber) != 16 {
			reject(w, http.StatusBadRequest, "05", "Invalid card number format")
			return
		}
		for _, ch := range req.CardNumber {
			if ch < '0' || ch > '9' {
				reject(w, http.StatusBadRequest, "05", "Invalid card number format")
				return
			}
		}

		card := db.FindCard(req.CardNumber)
		if card == nil {
			reject(w, http.StatusNotFound, "05", "Invalid card")
			return
		}

		// Fix #1: spec requires "05" for blocked card (not "62")
		if card.Status != models.Active {
			reject(w, http.StatusForbidden, "05", "Card is blocked")
			return
		}

		// Fix #2: validate type and amount BEFORE checking PIN
		// so the audit log only records transactions with valid fields
		if req.Type != models.Withdraw && req.Type != models.Topup {
			reject(w, http.StatusBadRequest, "12", "Invalid transaction type")
			return
		}

		if req.Amount <= 0 {
			reject(w, http.StatusBadRequest, "13", "Amount must be greater than zero")
			return
		}

		// Fix #7: round amount to 2 decimal places (payment standard)
		req.Amount = math.Round(req.Amount*100) / 100

		if store.HashPIN(req.Pin) != card.PinHash {
			stamp(db, req.CardNumber, req.Type, req.Amount, models.Failed)
			reject(w, http.StatusUnauthorized, "06", "Invalid PIN")
			return
		}

		// Fix #4: Withdraw now returns the new balance atomically
		var newBalance float64
		switch req.Type {
		case models.Withdraw:
			balance, ok := db.Withdraw(req.CardNumber, req.Amount)
			if !ok {
				stamp(db, req.CardNumber, req.Type, req.Amount, models.Failed)
				reject(w, http.StatusUnprocessableEntity, "99", "Insufficient balance")
				return
			}
			newBalance = balance
		case models.Topup:
			newBalance = db.Topup(req.CardNumber, req.Amount)
		}

		stamp(db, req.CardNumber, req.Type, req.Amount, models.Success)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.TransactionResponse{
			Status:   models.Success,
			RespCode: "00",
			Balance:  newBalance,
		})
	}
}

func Balance(db *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			reject(w, http.StatusMethodNotAllowed, "05", "method not allowed")
			return
		}

		num := strings.TrimPrefix(r.URL.Path, "/api/card/balance/")
		if num == "" {
			reject(w, http.StatusBadRequest, "05", "card number required")
			return
		}

		card := db.FindCard(num)
		if card == nil {
			reject(w, http.StatusNotFound, "05", "Card not found")
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.BalanceResponse{
			CardNumber: card.CardNumber,
			CardHolder: card.CardHolder,
			Balance:    card.Balance,
			Status:     card.Status,
		})
	}
}

func TxnHistory(db *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			reject(w, http.StatusMethodNotAllowed, "05", "method not allowed")
			return
		}

		num := strings.TrimPrefix(r.URL.Path, "/api/card/transactions/")
		if num == "" {
			reject(w, http.StatusBadRequest, "05", "card number required")
			return
		}

		if db.FindCard(num) == nil {
			reject(w, http.StatusNotFound, "05", "Card not found")
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(db.History(num))
	}
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "card-transaction-engine",
	})
}

func AllCards(db *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			reject(w, http.StatusMethodNotAllowed, "05", "method not allowed")
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(db.Cards())
	}
}

func reject(w http.ResponseWriter, status int, code, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Status:   string(models.Failed),
		RespCode: code,
		Message:  msg,
	})
}

func stamp(db *store.MemoryStore, num string, t models.TxnType, amt float64, s models.TxnStatus) {
	db.Record(&models.Transaction{
		TransactionID: db.NextID(),
		CardNumber:    num,
		Type:          t,
		Amount:        amt,
		Status:        s,
		Timestamp:     time.Now().UTC(),
	})
}