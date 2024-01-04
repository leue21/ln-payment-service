package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	qrcode "github.com/skip2/go-qrcode"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
)

func main() {
	apiKey := os.Getenv("API_KEY")
	walletId := os.Getenv("WALLET_ID")
	paymentUrl, err := url.Parse(os.Getenv("PAYMENT_URL"))
	if err != nil {
		log.Fatalln(err)
	}
	webhook, err := url.Parse(os.Getenv("WEBHOOK"))
	if err != nil {
		log.Fatalln(err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "API_KEY=%s,  webhook=%s, walletId=%s, paymentUrl=%s", apiKey, webhook, walletId, paymentUrl)
	listenAndServe(paymentUrl, apiKey, walletId, webhook)
}

func listenAndServe(paymentUrl *url.URL, apiKey string, walledId string, webhook *url.URL) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	mux := http.NewServeMux()
	mux.Handle("/", &PaymentHandler{
		p: &PaymentService{
			paymentUrl: paymentUrl,
			apiKey:     apiKey,
			webhook:    webhook,
			client:     client,
			walletId:   walledId,
		},
	})
	err := http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server running on :3000")
}

func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && regexp.MustCompile(`payment`).MatchString(r.URL.Path):
		h.CreatePayment(w, r)
		return
	case r.Method == http.MethodGet && regexp.MustCompile(`payment`).MatchString(r.URL.Path):
		h.GetPayment(w, r)
		return
	case r.Method == http.MethodPost && regexp.MustCompile(`paid`).MatchString(r.URL.Path):
		h.Paid(w, r)
		return
	case r.Method == http.MethodGet && regexp.MustCompile(`generate`).MatchString(r.URL.Path):
		h.GeneratePayment(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var pr PaymentRequest
	err := decoder.Decode(&pr)
	if err != nil {
		WriteStatusWithMessageHandler(w, http.StatusInternalServerError, "Internal Server Error")
	}
	defer closeBody(r)
	invoice, err := h.p.CreateInvoice(pr)
	if err != nil {
		WriteStatusWithMessageHandler(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(invoice)
}

func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	checkingId := r.URL.Query().Get("checking_id")
	if checkingId == "" {
		WriteStatusWithMessageHandler(w, http.StatusBadRequest, "checking_id is required")
		return
	}
	paid, err := h.p.CheckPayment(checkingId)
	if err != nil {
		WriteStatusWithMessageHandler(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(paid)
}

func (h *PaymentHandler) GeneratePayment(w http.ResponseWriter, r *http.Request) {
	amount := r.URL.Query().Get("amount")
	if amount == "" {
		WriteStatusWithMessageHandler(w, http.StatusBadRequest, "amount is required")
		return
	}
	s, err := strconv.Atoi(amount)
	if err != nil {
		WriteStatusWithMessageHandler(w, http.StatusBadRequest, "amount is not a number")
		return
	}
	invoice, err := h.p.CreateInvoice(PaymentRequest{
		Amount:   s,
		Currency: "sat",
	})
	qr := QRCode{
		Size:    256,
		Content: invoice.PaymentRequest,
	}
	qrCode, err := qr.Generate()
	if err != nil {
		WriteStatusWithMessageHandler(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(qrCode)
}

func (h *PaymentHandler) Paid(w http.ResponseWriter, r *http.Request) {
	fmt.Println("------------Paid------------")
	defer closeBody(r)
	decoder := json.NewDecoder(r.Body)
	var p Payment
	err := decoder.Decode(&p)
	if err != nil {
		WriteStatusWithMessageHandler(w, http.StatusInternalServerError, "Internal Server Error")
		return
	} else {
		err := h.p.Paid(p)
		if err != nil {
			WriteStatusWithMessageHandler(w, http.StatusInternalServerError, "Payment failed")
			return
		}
		WriteStatusWithMessageHandler(w, http.StatusOK, "OK")
	}
}

func closeBody(r *http.Request) {
	func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(r.Body)
}

func WriteStatusWithMessageHandler(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
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
	invoice, err := s.createPayment(InvoiceRequest{
		Out:     false,
		Amount:  pr.Amount,
		Memo:    "Payment",
		Expiry:  3600,
		Unit:    pr.Currency,
		Webhook: s.webhook.String(),
		//Webhook:  "https://webhook.site/218b74dd-19c5-4edd-85e7-c5806e254176",
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

	if err != nil {
		return Invoice{}, err
	}
	if r.StatusCode != http.StatusCreated {
		return Invoice{}, fmt.Errorf("status code is not 201")
	}
	defer closeBody(r.Request)
	decoder := json.NewDecoder(r.Body)
	var invoice Invoice
	err = decoder.Decode(&invoice)
	if err != nil {
		return Invoice{}, err
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
	fmt.Println("Paid")
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

func (qr *QRCode) Generate() ([]byte, error) {
	fmt.Println("Generate")
	qrCode, err := qrcode.Encode(qr.Content, qrcode.Medium, qr.Size)
	if err != nil {
		return nil, fmt.Errorf("could not generate a QR code: %v", err)
	}
	return qrCode, nil
}

// Path: handlers.go
type PaymentHandler struct {
	p *PaymentService
}

// Path: services.go
type PaymentService struct {
	paymentUrl *url.URL
	apiKey     string
	walletId   string
	webhook    *url.URL
	client     *http.Client
}

// Path: models.go
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
