package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SandeepXT/Card-Transaction-Engine/internal/models"
	"github.com/SandeepXT/Card-Transaction-Engine/internal/store"
)

// newDB returns a fresh in-memory store seeded with demo cards.
func newDB() *store.MemoryStore { return store.NewMemoryStore() }

// post is a helper that fires a POST /api/transaction and returns the recorder.
func post(t *testing.T, db *store.MemoryStore, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/transaction", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	Transaction(db)(rr, req)
	return rr
}

// --- Transaction endpoint ---

func TestTransaction_HappyWithdraw(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     200,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body)
	}
	var resp models.TransactionResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.RespCode != "00" {
		t.Errorf("expected respCode 00, got %s", resp.RespCode)
	}
	if resp.Balance != 800 {
		t.Errorf("expected balance 800, got %.2f", resp.Balance)
	}
}

func TestTransaction_HappyTopup(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "topup",
		"amount":     500,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp models.TransactionResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Balance != 1500 {
		t.Errorf("expected balance 1500, got %.2f", resp.Balance)
	}
}

func TestTransaction_InvalidCard(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "9999999999999999",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     100,
	})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	var resp models.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.RespCode != "05" {
		t.Errorf("expected respCode 05, got %s", resp.RespCode)
	}
}

func TestTransaction_BlockedCard(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4111111111111111",
		"pin":        "9999",
		"type":       "withdraw",
		"amount":     100,
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	var resp models.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	// Fix #1 verified: blocked card must return respCode "05" per spec
	if resp.RespCode != "05" {
		t.Errorf("expected respCode 05 for blocked card, got %s", resp.RespCode)
	}
}

func TestTransaction_WrongPIN(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "0000",
		"type":       "withdraw",
		"amount":     100,
	})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	var resp models.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.RespCode != "06" {
		t.Errorf("expected respCode 06, got %s", resp.RespCode)
	}
}

func TestTransaction_InsufficientBalance(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     9999,
	})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
	var resp models.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.RespCode != "99" {
		t.Errorf("expected respCode 99, got %s", resp.RespCode)
	}
}

func TestTransaction_InvalidType(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "refund",
		"amount":     100,
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var resp models.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.RespCode != "12" {
		t.Errorf("expected respCode 12, got %s", resp.RespCode)
	}
}

func TestTransaction_ZeroAmount(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     0,
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var resp models.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.RespCode != "13" {
		t.Errorf("expected respCode 13, got %s", resp.RespCode)
	}
}

func TestTransaction_InvalidCardFormat(t *testing.T) {
	db := newDB()
	// Fix #3 verified: short card number rejected before hitting the store
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "123",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     100,
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short card number, got %d", rr.Code)
	}
}

// Fix #2 verified: invalid type + wrong PIN should NOT create a log entry
func TestTransaction_InvalidTypeDoesNotLog(t *testing.T) {
	db := newDB()
	post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "0000", // wrong PIN
		"type":       "refund", // invalid type
		"amount":     100,
	})
	// Type validated first — request rejected at "12" before PIN is checked.
	// History must be empty.
	history := db.History("4123456789012345")
	if len(history) != 0 {
		t.Errorf("expected 0 log entries for invalid type, got %d", len(history))
	}
}

// Fix #7 verified: fractional amounts are rounded to 2 decimal places
func TestTransaction_AmountRounding(t *testing.T) {
	db := newDB()
	rr := post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     100.999, // should round to 101.00
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp models.TransactionResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	// 1000 - 101 = 899
	if resp.Balance != 899 {
		t.Errorf("expected balance 899 after rounded withdraw, got %.4f", resp.Balance)
	}
}

// --- Balance endpoint ---

func TestBalance_KnownCard(t *testing.T) {
	db := newDB()
	req := httptest.NewRequest(http.MethodGet, "/api/card/balance/4123456789012345", nil)
	rr := httptest.NewRecorder()
	Balance(db)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp models.BalanceResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Balance != 1000 {
		t.Errorf("expected 1000, got %.2f", resp.Balance)
	}
}

func TestBalance_UnknownCard(t *testing.T) {
	db := newDB()
	req := httptest.NewRequest(http.MethodGet, "/api/card/balance/0000000000000000", nil)
	rr := httptest.NewRecorder()
	Balance(db)(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

// --- TxnHistory endpoint ---

func TestTxnHistory_AfterWithdraw(t *testing.T) {
	db := newDB()
	post(t, db, map[string]interface{}{
		"cardNumber": "4123456789012345",
		"pin":        "1234",
		"type":       "withdraw",
		"amount":     100,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/card/transactions/4123456789012345", nil)
	rr := httptest.NewRecorder()
	TxnHistory(db)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var txns []*models.Transaction
	json.NewDecoder(rr.Body).Decode(&txns)
	if len(txns) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(txns))
	}
	if txns[0].Status != models.Success {
		t.Errorf("expected SUCCESS, got %s", txns[0].Status)
	}
}

// --- Health endpoint ---

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}