package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/skip2/go-qrcode"
	"log"
	"net/http"
	"net/url"
)

type PaymentService struct {
	paymentUrl *url.URL
	apiKey     string
	walletId   string
	webhook    *url.URL
	successUrl *url.URL
	client     *http.Client
}

func (qr *QRCode) GenerateQRCode() ([]byte, error) {
	fmt.Println("Generate")
	qrCode, err := qrcode.Encode(qr.Content, qrcode.Medium, qr.Size)
	if err != nil {
		return nil, fmt.Errorf("could not generate a QR code: %v", err)
	}
	return qrCode, nil
}

func (s *PaymentService) CreateInvoice(pr PaymentRequest) (Invoice, error) {
	if pr.Amount == 0 {
		return Invoice{}, fmt.Errorf("amount is required")
	}
	if pr.Currency == "" {
		return Invoice{}, fmt.Errorf("currency is required")
	}
	if pr.Currency != "sat" {
		return Invoice{}, fmt.Errorf("currency is not supported yet")
	}
	memo := fmt.Sprintf("Invoice for %s", pr.Item)
	invoice, err := s.createPayment(InvoiceRequest{
		Out:      false,
		Amount:   pr.Amount,
		Memo:     memo,
		Expiry:   3600,
		Unit:     pr.Currency,
		Webhook:  s.webhook.String(),
		Internal: false,
	})
	if err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func (s *PaymentService) createPayment(ir InvoiceRequest) (Invoice, error) {
	jsonBody, err := json.Marshal(ir)
	if err != nil {
		return Invoice{}, err
	}
	req, err := http.NewRequest("POST", s.paymentUrl.String(), bytes.NewReader(jsonBody))
	req.Header.Set("X-Api-Key", s.apiKey)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	r, err := s.client.Do(req)
	empty := Invoice{}
	if err != nil {
		return empty, err
	}
	if r.StatusCode != http.StatusCreated {
		return empty, fmt.Errorf("status code is not 201")
	}
	defer closeBody(r.Request)
	decoder := json.NewDecoder(r.Body)
	var invoice Invoice
	err = decoder.Decode(&invoice)
	if err != nil {
		return empty, err
	}
	return invoice, nil
}

func (s *PaymentService) CheckPayment(checkingId string) ([]Payment, error) {
	req, err := http.NewRequest("GET", s.paymentUrl.String(), nil)
	if err != nil {
		return []Payment{}, err
	}
	values := req.URL.Query()
	values.Add("checking_id", checkingId)
	req.URL.RawQuery = values.Encode()
	req.Header.Set("X-Api-Key", s.apiKey)
	req.Header.Add("Accept", "application/json")
	r, err := s.client.Do(req)
	if err != nil {
		return []Payment{}, err
	}
	if r.StatusCode != http.StatusOK {
		return []Payment{}, fmt.Errorf("status code is not 200")
	}
	decoder := json.NewDecoder(r.Body)
	var payments []Payment
	err = decoder.Decode(&payments)
	if err != nil {
		return []Payment{}, err
	}
	return payments, nil
}

func (s *PaymentService) Paid(payment Payment) error {
	fmt.Printf("Payment: %v\n", payment)
	err := s.validatePayment(payment)
	if err != nil {
		return err
	}
	a := PaymentAction{
		Action:     "blink",
		CheckingId: payment.CheckingId,
		Amount:     10,
	}
	jsonBody, err := json.Marshal(a)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.successUrl.String(), bytes.NewReader(jsonBody))
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		log.Fatal("status code is not 200")
		return err
	}
	return nil
}

func (s *PaymentService) validatePayment(payment Payment) error {
	if payment.Amount == 0 {
		return fmt.Errorf("amount is required")
	}
	if payment.CheckingId == "" {
		return fmt.Errorf("checking_id is required")
	}
	checkPayment, err := s.CheckPayment(payment.CheckingId)
	if err != nil {
		return err
	}
	if len(checkPayment) == 0 {
		return fmt.Errorf("payment not found")
	}
	if checkPayment[0].Pending {
		return fmt.Errorf("payment is pending")
	}
	return nil
}

func (s *PaymentService) getJson(url string, target interface{}) error {
	r, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer closeBody(r.Request)
	return json.NewDecoder(r.Body).Decode(target)
}
