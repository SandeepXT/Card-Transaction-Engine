package models

import "time"

type CardStatus string

const (
	Active  CardStatus = "ACTIVE"
	Blocked CardStatus = "BLOCKED"
)

type TxnType string

const (
	Withdraw TxnType = "withdraw"
	Topup    TxnType = "topup"
)

type TxnStatus string

const (
	Success TxnStatus = "SUCCESS"
	Failed  TxnStatus = "FAILED"
)

type Card struct {
	CardNumber string     `json:"cardNumber"`
	CardHolder string     `json:"cardHolder"`
	PinHash    string     `json:"-"`
	Balance    float64    `json:"balance"`
	Status     CardStatus `json:"status"`
}

type Transaction struct {
	TransactionID string    `json:"transactionId"`
	CardNumber    string    `json:"cardNumber"`
	Type          TxnType   `json:"type"`
	Amount        float64   `json:"amount"`
	Status        TxnStatus `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

type TransactionRequest struct {
	CardNumber string  `json:"cardNumber"`
	Pin        string  `json:"pin"`
	Type       TxnType `json:"type"`
	Amount     float64 `json:"amount"`
}

type TransactionResponse struct {
	Status   TxnStatus `json:"status"`
	RespCode string    `json:"respCode"`
	Balance  float64   `json:"balance,omitempty"`
	Message  string    `json:"message,omitempty"`
}

type BalanceResponse struct {
	CardNumber string     `json:"cardNumber"`
	CardHolder string     `json:"cardHolder"`
	Balance    float64    `json:"balance"`
	Status     CardStatus `json:"status"`
}

type ErrorResponse struct {
	Status   string `json:"status"`
	RespCode string `json:"respCode"`
	Message  string `json:"message"`
}
