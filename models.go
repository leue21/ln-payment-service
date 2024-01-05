package main

import (
	"fmt"
	"github.com/skip2/go-qrcode"
)

type Invoice struct {
	PaymentHash    string `json:"payment_hash"`
	PaymentRequest string `json:"payment_request"`
	CheckingId     string `json:"checking_id"`
}

type PaymentRequest struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type InvoiceRequest struct {
	Out      bool   `json:"out"`
	Amount   int    `json:"amount"`
	Memo     string `json:"memo"`
	Expiry   int    `json:"expiry"`
	Unit     string `json:"unit"`
	Webhook  string `json:"webhook"`
	Internal bool   `json:"internal"`
}

type Payment struct {
	CheckingId    string `json:"checking_id"`
	Pending       bool   `json:"pending"`
	Amount        int    `json:"amount"`
	Fee           int    `json:"fee"`
	Memo          string `json:"memo"`
	Time          int    `json:"time"`
	Bolt11        string `json:"bolt11"`
	Preimage      string `json:"preimage"`
	PaymentHash   string `json:"payment_hash"`
	WalletId      string `json:"wallet_id"`
	Webhook       string `json:"webhook"`
	WebhookStatus int    `json:"webhook_status"`
}

type QRCode struct {
	Size    int
	Content string
}

func (qr *QRCode) Generate() ([]byte, error) {
	qrCode, err := qrcode.Encode(qr.Content, qrcode.Medium, qr.Size)
	if err != nil {
		return nil, fmt.Errorf("could not generate a QR code: %v", err)
	}
	return qrCode, nil
}
