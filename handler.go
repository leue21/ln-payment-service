package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

type PaymentHandler struct {
	p *PaymentService
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
