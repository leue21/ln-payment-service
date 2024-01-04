package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
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
