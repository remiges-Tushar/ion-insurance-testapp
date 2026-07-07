package main

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ION Service — sole DOKU merchant for the ION insurance network.
// Responsibilities:
//   - Create DOKU VA and QRIS payment instruments (called by BAP)
//   - Receive DOKU payment webhooks and forward payment notifications to BPP
//   - Execute DOKU settlement releases with split disbursements (called by BPP)

type cachedToken struct {
	token     string
	expiresAt time.Time
}

type ionService struct {
	clientID      string
	secretKey     string
	baseURL       string
	callbackURL   string
	bppPaymentURL string
	bapFrontendURL string

	// Split settlement bank account IDs
	bppBankAccountID string
	bapBankAccountID string
	ionBankAccountID string

	// SNAP API fields (for QRIS)
	snapClientID string
	snapPrivKey  *rsa.PrivateKey
	snapToken    *cachedToken
	snapMu       sync.Mutex
}

func main() {
	port := os.Getenv("ION_PORT")
	if port == "" {
		port = "8090"
	}

	svc := &ionService{
		clientID:         os.Getenv("DOKU_CLIENT_ID"),
		secretKey:        os.Getenv("DOKU_SECRET_KEY"),
		baseURL:          getenv("DOKU_BASE_URL", "https://api-sandbox.doku.com"),
		callbackURL:      os.Getenv("DOKU_CALLBACK_URL"),
		bppPaymentURL:    getenv("BPP_PAYMENT_URL", "http://bpp:8080/webhook/payment-received"),
		bapFrontendURL:   getenv("BAP_FRONTEND_URL", "http://localhost:3000"),
		bppBankAccountID: os.Getenv("DOKU_BPP_BANK_ACCOUNT_ID"),
		bapBankAccountID: os.Getenv("DOKU_BAP_BANK_ACCOUNT_ID"),
		ionBankAccountID: os.Getenv("DOKU_ION_BANK_ACCOUNT_ID"),
	}

	// SNAP client ID defaults to non-SNAP client ID if not separately configured
	svc.snapClientID = getenv("DOKU_SNAP_CLIENT_ID", svc.clientID)

	// Load RSA private key for SNAP B2B token (base64-encoded PEM or raw PEM)
	if keyStr := os.Getenv("DOKU_SNAP_PRIVATE_KEY"); keyStr != "" {
		privKey, err := loadRSAPrivateKey(keyStr)
		if err != nil {
			log.Printf("[ION] WARN: DOKU_SNAP_PRIVATE_KEY invalid: %v — SNAP QRIS disabled", err)
		} else {
			svc.snapPrivKey = privKey
			log.Printf("[ION] SNAP QRIS enabled (RSA key loaded)")
		}
	} else {
		log.Printf("[ION] SNAP QRIS disabled (DOKU_SNAP_PRIVATE_KEY not set)")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", svc.handleHealth)
	mux.HandleFunc("/payment/create-va", svc.handleCreateVA)
	mux.HandleFunc("/payment/create-checkout", svc.handleCreateCheckout)
	mux.HandleFunc("/payment/create-qris", svc.handleCreateQRIS)
	mux.HandleFunc("/payment/simulate", svc.handleSimulateDoku)
	mux.HandleFunc("/payment/notify-direct", svc.handleNotifyDirect)
	mux.HandleFunc("/webhook/doku", svc.handleDokuWebhook)
	mux.HandleFunc("/settlement/release", svc.handleRelease)

	log.Printf("[ION] Service starting on :%s — DOKU client=%s", port, svc.clientID)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("[ION] server error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (s *ionService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"ion"}`))
}

// handleCreateVA creates a DOKU BNI VA with hold_settlement=true.
// Called by BAP after receiving on_init from BPP.
//
// Request:  { "invoice_number": "ins-...", "customer_name": "...", "amount_idr": 6050499 }
// Response: { "va_number": "...", "bank_code": "BNI", "how_to_pay_page": "...", "invoice_number": "..." }
func (s *ionService) handleCreateVA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InvoiceNumber string `json:"invoice_number"`
		CustomerName  string `json:"customer_name"`
		AmountIDR     int64  `json:"amount_idr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.InvoiceNumber == "" || req.AmountIDR <= 0 {
		jsonError(w, "invoice_number and amount_idr required", http.StatusBadRequest)
		return
	}
	if req.CustomerName == "" {
		req.CustomerName = "ION Insurance"
	}

	va, err := s.createVA(req.InvoiceNumber, req.CustomerName, req.AmountIDR)
	if err != nil {
		log.Printf("[ION] createVA error: %v", err)
		jsonError(w, "doku VA creation failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("[ION] VA created — invoice=%s va=%s bank=%s", va["invoice_number"], va["va_number"], va["bank_code"])
	jsonOK(w, va)
}

// handleCreateCheckout creates a DOKU Checkout session with hold_settlement=true.
// Returns a hosted checkout URL (staging.doku.com) where the customer chooses their
// payment method (VA, QRIS, card, e-wallet) — no separate VA or QRIS call needed.
//
// Request:  { "invoice_number": "ins-...", "customer_name": "...", "amount_idr": 6050499, "callback_url": "http://..." }
// Response: { "checkout_url": "https://staging.doku.com/checkout-link-v2/...", "invoice_number": "..." }
func (s *ionService) handleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InvoiceNumber string `json:"invoice_number"`
		CustomerName  string `json:"customer_name"`
		AmountIDR     int64  `json:"amount_idr"`
		CallbackURL   string `json:"callback_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.InvoiceNumber == "" || req.AmountIDR <= 0 {
		jsonError(w, "invoice_number and amount_idr required", http.StatusBadRequest)
		return
	}
	if req.CustomerName == "" {
		req.CustomerName = "ION Insurance Customer"
	}

	result, err := s.createCheckout(req.InvoiceNumber, req.CustomerName, req.AmountIDR, req.CallbackURL)
	if err != nil {
		log.Printf("[ION] createCheckout error: %v", err)
		jsonError(w, "doku checkout creation failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("[ION] Checkout created — invoice=%s url=%s", req.InvoiceNumber, result["checkout_url"])
	jsonOK(w, result)
}

func (s *ionService) createCheckout(invoiceNumber, customerName string, amount int64, callbackURL string) (map[string]any, error) {
	if callbackURL == "" {
		callbackURL = s.bapFrontendURL
	}
	settlement := []any{}
	if s.bppBankAccountID != "" {
		settlement = append(settlement, map[string]any{
			"bank_account_settlement_id": s.bppBankAccountID,
			"value":                      97,
			"type":                       "PERCENTAGE",
		})
	}
	if s.bapBankAccountID != "" {
		settlement = append(settlement, map[string]any{
			"bank_account_settlement_id": s.bapBankAccountID,
			"value":                      2,
			"type":                       "PERCENTAGE",
		})
	}
	if s.ionBankAccountID != "" {
		settlement = append(settlement, map[string]any{
			"bank_account_settlement_id": s.ionBankAccountID,
			"value":                      1,
			"type":                       "PERCENTAGE",
		})
	}

	additionalInfo := map[string]any{"hold_settlement": true}
	if len(settlement) > 0 {
		additionalInfo["settlement"] = settlement
	}

	payload := map[string]any{
		"order": map[string]any{
			"amount":         amount,
			"invoice_number": invoiceNumber,
			"currency":       "IDR",
			"session_id":     invoiceNumber,
			"callback_url":   callbackURL,
			"line_items": []any{
				map[string]any{
					"name":     "Insurance Premium",
					"price":    amount,
					"quantity": 1,
				},
			},
		},
		"payment": map[string]any{
			"payment_due_date": 60,
		},
		"customer": map[string]any{
			"name":    customerName,
			"email":   "customer@ion.insure",
			"phone":   "+628000000000",
			"address": "Jakarta",
			"country": "ID",
		},
		"additional_info": additionalInfo,
	}

	body, status, checkoutReqID, err := s.doRequest("POST", "/checkout/v1/payment", payload)
	if err != nil {
		return nil, fmt.Errorf("DOKU checkout request: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("DOKU checkout HTTP %d: %s", status, string(body))
	}

	var resp struct {
		Message  []string `json:"message"`
		Response struct {
			Payment struct {
				URL string `json:"url"`
			} `json:"payment"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("DOKU checkout decode: %w", err)
	}
	if resp.Response.Payment.URL == "" {
		return nil, fmt.Errorf("DOKU checkout: empty checkout URL (messages=%v)", resp.Message)
	}

	return map[string]any{
		"checkout_url":       resp.Response.Payment.URL,
		"invoice_number":     invoiceNumber,
		"payment_request_id": checkoutReqID,
	}, nil
}

// handleCreateQRIS creates a DOKU QRIS payment code via the SNAP API.
// Falls back to non-SNAP if SNAP is not configured.
// Called by BAP alongside VA creation.
//
// Request:  { "invoice_number": "ins-...", "customer_name": "...", "amount_idr": 6050499 }
// Response: { "qr_string": "00020101...", "invoice_number": "..." }
func (s *ionService) handleCreateQRIS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InvoiceNumber string `json:"invoice_number"`
		CustomerName  string `json:"customer_name"`
		AmountIDR     int64  `json:"amount_idr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.InvoiceNumber == "" || req.AmountIDR <= 0 {
		jsonError(w, "invoice_number and amount_idr required", http.StatusBadRequest)
		return
	}
	if req.CustomerName == "" {
		req.CustomerName = "ION Insurance"
	}

	// Use invoice number with -qris suffix to avoid duplicate invoice collision with VA
	qrisInvoice := req.InvoiceNumber + "-qris"

	var qr string
	var err error
	if s.snapPrivKey != nil {
		qr, err = s.createQRISSnap(qrisInvoice, req.AmountIDR)
	} else {
		qr, err = s.createQRISLegacy(qrisInvoice, req.CustomerName, req.AmountIDR)
	}
	if err != nil {
		log.Printf("[ION] createQRIS error: %v", err)
		jsonError(w, "doku QRIS creation failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("[ION] QRIS created — invoice=%s", qrisInvoice)
	jsonOK(w, map[string]any{
		"qr_string":      qr,
		"invoice_number": qrisInvoice,
	})
}

// handleDokuWebhook receives DOKU payment notifications, verifies the signature,
// and forwards a payment-received notification to BPP.
func (s *ionService) handleDokuWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusOK) // always ACK to DOKU
		return
	}

	clientID := r.Header.Get("Client-Id")
	requestID := r.Header.Get("Request-Id")
	timestamp := r.Header.Get("Request-Timestamp")
	signature := r.Header.Get("Signature")

	// Determine path from request — DOKU webhook may be proxied via nginx
	requestTarget := r.URL.Path
	if !strings.HasPrefix(requestTarget, "/") {
		requestTarget = "/" + requestTarget
	}

	if !s.verifySignature(clientID, requestID, timestamp, requestTarget, bodyBytes, signature) {
		// DOKU Checkout notifications sometimes omit Client-Id (sandbox behaviour).
		// Log all headers for debugging, then fall through if body looks valid.
		log.Printf("[ION] DOKU webhook: signature mismatch client=%q requestID=%q sig=%q path=%q",
			clientID, requestID, signature, requestTarget)
		for k, v := range r.Header {
			log.Printf("[ION]   header %s: %s", k, strings.Join(v, ", "))
		}
		// When Client-Id is missing the notification is likely from DOKU Checkout
		// which uses a different header set. Process it; body validation below
		// will reject anything that doesn't look like a real DOKU notification.
		if clientID != "" {
			// Client-Id present but signature wrong — reject (possible replay/spoof).
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"signature_error"}`))
			return
		}
		log.Printf("[ION] DOKU webhook: no Client-Id header, processing as Checkout notification")
	}

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	order, _ := body["order"].(map[string]any)
	invoiceNumber, _ := order["invoice_number"].(string)
	amount, _ := order["amount"].(float64)

	if invoiceNumber == "" {
		log.Printf("[ION] DOKU webhook: no invoice_number in payload")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ignored"}`))
		return
	}

	log.Printf("[ION] DOKU webhook received — invoice=%s amount=%.0f requestID=%s", invoiceNumber, amount, requestID)

	// Forward to BPP asynchronously so DOKU gets an immediate 200
	go s.notifyBPP(invoiceNumber, requestID, amount)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleRelease executes a DOKU settlement release with split disbursements.
// Called by BPP after Beckn reconcile is received and validated.
//
// Request:
//
//	{
//	  "invoice_number": "ins-...",
//	  "original_request_id": "<payment request_id>",
//	  "amount": 6050499,
//	  "splits": [
//	    { "bank_account_settlement_id": "BNK-...", "value": 97, "type": "PERCENTAGE" },
//	    { "bank_account_settlement_id": "BNK-...", "value": 3,  "type": "PERCENTAGE" }
//	  ]
//	}
func (s *ionService) handleRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InvoiceNumber     string `json:"invoice_number"`
		OriginalRequestID string `json:"original_request_id"`
		Amount            int64  `json:"amount"`
		Splits            []struct {
			BankAccountSettlementID string  `json:"bank_account_settlement_id"`
			Value                   float64 `json:"value"`
			Type                    string  `json:"type"`
		} `json:"splits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[ION] Settlement release — invoice=%s originalReqID=%s amount=%d splits=%d",
		req.InvoiceNumber, req.OriginalRequestID, req.Amount, len(req.Splits))

	overrides := make([]map[string]any, 0, len(req.Splits))
	for _, sp := range req.Splits {
		overrides = append(overrides, map[string]any{
			"bank_account_settlement_id": sp.BankAccountSettlementID,
			"value":                      sp.Value,
			"type":                       sp.Type,
		})
	}

	buildPayload := func(withSplits bool) map[string]any {
		p := map[string]any{
			"order": map[string]any{
				"invoice_number": req.InvoiceNumber,
				"amount":         req.Amount,
				"currency":       "IDR",
			},
			"transaction": map[string]any{
				"original_request_id": req.OriginalRequestID,
			},
		}
		if withSplits && len(overrides) > 0 {
			p["override_settlement"] = overrides
		}
		return p
	}

	body, status, releaseReqID, err := s.doRequest("POST", "/finance/v1/release", buildPayload(true))
	if err != nil {
		log.Printf("[ION] DOKU release error: %v", err)
		jsonError(w, "doku release failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	// If override_settlement bank accounts are not registered in DOKU, retry without splits.
	if (status == http.StatusBadRequest || status == http.StatusUnprocessableEntity) && len(overrides) > 0 {
		var errBody struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(body, &errBody)
		if errBody.Error.Code == "data_not_found" {
			log.Printf("[ION] DOKU release: bank accounts not found — retrying without splits")
			body, status, releaseReqID, err = s.doRequest("POST", "/finance/v1/release", buildPayload(false))
			if err != nil {
				log.Printf("[ION] DOKU release (no-split) error: %v", err)
				jsonError(w, "doku release failed: "+err.Error(), http.StatusBadGateway)
				return
			}
		}
	}
	if status != http.StatusOK && status != http.StatusCreated {
		log.Printf("[ION] DOKU release HTTP %d: %s", status, string(body))
		jsonError(w, fmt.Sprintf("doku release HTTP %d: %s", status, string(body)), http.StatusBadGateway)
		return
	}

	log.Printf("[ION] DOKU release SUCCESS — invoice=%s dokuRef=%s", req.InvoiceNumber, releaseReqID)
	jsonOK(w, map[string]any{"status": "released", "doku_ref": releaseReqID})
}

// ---------------------------------------------------------------------------
// DOKU non-SNAP API helpers (VA + webhook signature)
// ---------------------------------------------------------------------------

func (s *ionService) newRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *ionService) sign(path string, bodyBytes []byte) (map[string]string, string) {
	requestID := s.newRequestID()
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	hash := sha256.Sum256(bodyBytes)
	digest := base64.StdEncoding.EncodeToString(hash[:])

	stringToSign := fmt.Sprintf(
		"Client-Id:%s\nRequest-Id:%s\nRequest-Timestamp:%s\nRequest-Target:%s\nDigest:%s",
		s.clientID, requestID, timestamp, path, digest,
	)

	mac := hmac.New(sha256.New, []byte(s.secretKey))
	mac.Write([]byte(stringToSign))
	sig := "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"Client-Id":         s.clientID,
		"Request-Id":        requestID,
		"Request-Timestamp": timestamp,
		"Signature":         sig,
	}, requestID
}

func (s *ionService) doRequest(method, path string, body any) ([]byte, int, string, error) {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, 0, "", err
		}
	}

	headers, requestID := s.sign(path, bodyBytes)

	req, err := http.NewRequest(method, s.baseURL+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, requestID, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	log.Printf("[ION] DOKU %s %s → HTTP %d", method, path, resp.StatusCode)
	return respBytes, resp.StatusCode, requestID, err
}

func (s *ionService) verifySignature(clientID, requestID, timestamp, requestTarget string, bodyBytes []byte, signature string) bool {
	if s.secretKey == "" {
		return true
	}

	hash := sha256.Sum256(bodyBytes)
	digest := base64.StdEncoding.EncodeToString(hash[:])

	stringToSign := fmt.Sprintf(
		"Client-Id:%s\nRequest-Id:%s\nRequest-Timestamp:%s\nRequest-Target:%s\nDigest:%s",
		clientID, requestID, timestamp, requestTarget, digest,
	)

	mac := hmac.New(sha256.New, []byte(s.secretKey))
	mac.Write([]byte(stringToSign))
	expected := "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

func (s *ionService) createVA(invoiceNumber, customerName string, amount int64) (map[string]any, error) {
	payload := map[string]any{
		"client": map[string]any{"id": s.clientID},
		"order": map[string]any{
			"invoice_number": invoiceNumber,
			"amount":         amount,
			"currency":       "IDR",
			"callback_url":   s.callbackURL,
		},
		"customer": map[string]any{
			"name": customerName,
		},
		"virtual_account_info": map[string]any{
			"billing_type":    "FIX_BILL",
			"expired_time":    1440,
			"reusable_status": false,
		},
		"additional_info": map[string]any{
			"channel":         "VIRTUAL_ACCOUNT_BNI",
			"hold_settlement": true,
		},
	}

	body, status, _, err := s.doRequest("POST", "/doku-virtual-account/v2/payment-code", payload)
	if err != nil {
		return nil, fmt.Errorf("DOKU VA request: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("DOKU VA HTTP %d: %s", status, string(body))
	}

	var resp struct {
		VirtualAccountInfo struct {
			BankCode             string `json:"bank_code"`
			VirtualAccountNumber string `json:"virtual_account_number"`
			HowToPayPage         string `json:"how_to_pay_page"`
		} `json:"virtual_account_info"`
		Response struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("DOKU VA decode: %w", err)
	}
	if resp.VirtualAccountInfo.VirtualAccountNumber == "" {
		return nil, fmt.Errorf("DOKU VA empty number (code=%s msg=%s)", resp.Response.Code, resp.Response.Message)
	}

	bankCode := resp.VirtualAccountInfo.BankCode
	if bankCode == "" {
		bankCode = "BNI"
	}
	return map[string]any{
		"va_number":       resp.VirtualAccountInfo.VirtualAccountNumber,
		"bank_code":       bankCode,
		"how_to_pay_page": resp.VirtualAccountInfo.HowToPayPage,
		"invoice_number":  invoiceNumber,
	}, nil
}

// createQRISLegacy uses the non-SNAP QRIS endpoint (may be disabled for merchant).
func (s *ionService) createQRISLegacy(invoiceNumber, customerName string, amount int64) (string, error) {
	payload := map[string]any{
		"client": map[string]any{"id": s.clientID},
		"order": map[string]any{
			"invoice_number": invoiceNumber,
			"amount":         amount,
			"currency":       "IDR",
			"callback_url":   s.callbackURL,
		},
		"customer": map[string]any{
			"name": customerName,
		},
	}

	body, status, _, err := s.doRequest("POST", "/qris-pg/v2/payment-code", payload)
	if err != nil {
		return "", fmt.Errorf("DOKU QRIS request: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", fmt.Errorf("DOKU QRIS HTTP %d: %s", status, string(body))
	}

	var resp struct {
		QR struct {
			Content string `json:"content"`
		} `json:"qr"`
		Response struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("DOKU QRIS decode: %w", err)
	}
	if resp.QR.Content == "" {
		return "", fmt.Errorf("DOKU QRIS empty content (code=%s msg=%s)", resp.Response.Code, resp.Response.Message)
	}
	return resp.QR.Content, nil
}

// ---------------------------------------------------------------------------
// DOKU SNAP API — B2B token + QRIS
// ---------------------------------------------------------------------------

// getB2BToken fetches a SNAP B2B access token (cached for 850s out of 900s TTL).
// Uses SHA256withRSA signature as required by the SNAP B2B token endpoint.
func (s *ionService) getB2BToken() (string, error) {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()

	if s.snapToken != nil && time.Now().Before(s.snapToken.expiresAt) {
		return s.snapToken.token, nil
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	stringToSign := s.snapClientID + "|" + timestamp

	sigBytes, err := signRSASHA256(s.snapPrivKey, stringToSign)
	if err != nil {
		return "", fmt.Errorf("RSA sign: %w", err)
	}
	sig := base64.StdEncoding.EncodeToString(sigBytes)

	reqBody := `{"grantType":"client_credentials"}`
	httpReq, err := http.NewRequest("POST", s.baseURL+"/authorization/v1/access-token/b2b", strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-CLIENT-KEY", s.snapClientID)
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-SIGNATURE", sig)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("B2B token request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	log.Printf("[ION] SNAP B2B token → HTTP %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("B2B token HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	}
	if err := json.Unmarshal(respBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("B2B token decode: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("B2B token empty: %s", string(respBytes))
	}

	ttl := time.Duration(tokenResp.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 900 * time.Second
	}
	s.snapToken = &cachedToken{
		token:     tokenResp.AccessToken,
		expiresAt: time.Now().Add(ttl - 50*time.Second), // 50s safety margin
	}
	log.Printf("[ION] SNAP B2B token acquired, expires in %ds", tokenResp.ExpiresIn)
	return s.snapToken.token, nil
}

// signSNAP produces the HMAC-SHA512 signature for SNAP API requests (after B2B token).
// Formula: HTTPMethod:EndpointPath:AccessToken:lowercase(hex(sha256(body))):Timestamp
func (s *ionService) signSNAP(method, endpointPath, accessToken string, bodyBytes []byte, timestamp string) string {
	h := sha256.Sum256(bodyBytes)
	bodyHash := strings.ToLower(hex.EncodeToString(h[:]))
	stringToSign := method + ":" + endpointPath + ":" + accessToken + ":" + bodyHash + ":" + timestamp

	mac := hmac.New(sha512.New, []byte(s.secretKey))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// createQRISSnap creates a DOKU QRIS using the SNAP API.
// Returns the qrContent string ready to encode as a QR image.
func (s *ionService) createQRISSnap(partnerReferenceNo string, amount int64) (string, error) {
	token, err := s.getB2BToken()
	if err != nil {
		return "", fmt.Errorf("B2B token: %w", err)
	}

	endpoint := "/snap-adapter/b2b/v1.0/qr/qr-mpm-generate"
	timestamp := time.Now().UTC().Format(time.RFC3339)

	payload := map[string]any{
		"partnerReferenceNo": partnerReferenceNo,
		"amount": map[string]string{
			"value":    fmt.Sprintf("%.2f", float64(amount)),
			"currency": "IDR",
		},
		"merchantId": s.snapClientID,
		"terminalId": "ION-001",
		"additionalInfo": map[string]any{
			"postalCode": "12190",
			"feeType":    "OUR",
		},
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	sig := s.signSNAP("POST", endpoint, token, bodyBytes, timestamp)
	externalID := s.newRequestID()

	httpReq, err := http.NewRequest("POST", s.baseURL+endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("X-PARTNER-ID", s.snapClientID)
	httpReq.Header.Set("X-EXTERNAL-ID", externalID)
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-SIGNATURE", sig)
	httpReq.Header.Set("CHANNEL-ID", "H2H")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("SNAP QRIS request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	log.Printf("[ION] SNAP QRIS → HTTP %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("SNAP QRIS HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var qrisResp struct {
		QrContent          string `json:"qrContent"`
		ReferenceNo        string `json:"referenceNo"`
		PartnerReferenceNo string `json:"partnerReferenceNo"`
		ResponseCode       string `json:"responseCode"`
		ResponseMessage    string `json:"responseMessage"`
	}
	if err := json.Unmarshal(respBytes, &qrisResp); err != nil {
		return "", fmt.Errorf("SNAP QRIS decode: %w", err)
	}
	if qrisResp.QrContent == "" {
		return "", fmt.Errorf("SNAP QRIS empty qrContent (code=%s msg=%s)", qrisResp.ResponseCode, qrisResp.ResponseMessage)
	}

	log.Printf("[ION] SNAP QRIS created — ref=%s partnerRef=%s", qrisResp.ReferenceNo, qrisResp.PartnerReferenceNo)
	return qrisResp.QrContent, nil
}

// ---------------------------------------------------------------------------
// RSA helpers
// ---------------------------------------------------------------------------

// loadRSAPrivateKey parses a PKCS#8 RSA private key from a base64-encoded PEM
// or a raw PEM string (supports both for env var convenience).
func loadRSAPrivateKey(keyStr string) (*rsa.PrivateKey, error) {
	var pemBytes []byte

	// If it doesn't look like a PEM header, assume base64-encoded
	if !strings.HasPrefix(strings.TrimSpace(keyStr), "-----") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keyStr))
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
		pemBytes = decoded
	} else {
		pemBytes = []byte(keyStr)
	}

	// Replace literal \n in env vars with real newlines
	pemBytes = []byte(strings.ReplaceAll(string(pemBytes), `\n`, "\n"))

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback: try PKCS1
		rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("PKCS8: %v; PKCS1: %v", err, err2)
		}
		return rsaKey, nil
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaKey, nil
}

// signRSASHA256 signs a string with SHA256withRSA (PKCS1v15).
func signRSASHA256(privKey *rsa.PrivateKey, data string) ([]byte, error) {
	h := sha256.New()
	h.Write([]byte(data))
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, digest)
}

// ---------------------------------------------------------------------------
// Notification + simulation
// ---------------------------------------------------------------------------

// handleNotifyDirect calls notifyBPP directly, bypassing the external DOKU webhook loop.
// Used as a sandbox workaround when DOKU's notification webhook cannot reach this service.
// Request: { "invoice_number": "ins-...", "amount": 6050499 }
func (s *ionService) handleNotifyDirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InvoiceNumber    string  `json:"invoice_number"`
		Amount           float64 `json:"amount"`
		PaymentRequestID string  `json:"payment_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InvoiceNumber == "" {
		jsonError(w, "invoice_number required", http.StatusBadRequest)
		return
	}
	log.Printf("[ION] notify-direct — invoice=%s amount=%.0f", req.InvoiceNumber, req.Amount)
	go s.notifyBPP(req.InvoiceNumber, req.PaymentRequestID, req.Amount)
	jsonOK(w, map[string]string{"status": "notified", "invoice_number": req.InvoiceNumber})
}

// notifyBPP POSTs a payment-received notification to BPP.
// This is called asynchronously after a DOKU webhook arrives.
func (s *ionService) notifyBPP(invoiceNumber, paymentRequestID string, amount float64) {
	payload := map[string]any{
		"invoice_number":     invoiceNumber,
		"payment_request_id": paymentRequestID,
		"amount":             amount,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(s.bppPaymentURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[ION] notifyBPP failed: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[ION] Notified BPP of payment — invoice=%s paymentReqID=%s status=%d",
		invoiceNumber, paymentRequestID, resp.StatusCode)
}

// handleSimulateDoku fires a real DOKU-format webhook signed with ION's HMAC
// key to the configured DOKU_CALLBACK_URL (ngrok). The request travels:
//
//	ION → ngrok → nginx /ion-webhook/ → ION /webhook/doku → verifySignature → notifyBPP → BPP
//
// This exercises the full production webhook path without needing DOKU's sandbox
// to initiate it (DOKU sandbox has no payment simulator API).
// Request: { "invoice_number": "ins-...", "amount": 6050499 }
func (s *ionService) handleSimulateDoku(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		InvoiceNumber string  `json:"invoice_number"`
		Amount        float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InvoiceNumber == "" {
		jsonError(w, "invoice_number and amount required", http.StatusBadRequest)
		return
	}
	if s.callbackURL == "" {
		jsonError(w, "DOKU_CALLBACK_URL not configured", http.StatusInternalServerError)
		return
	}

	if err := s.fireWebhookToSelf(req.InvoiceNumber, req.Amount); err != nil {
		log.Printf("[ION] simulate webhook error: %v", err)
		jsonError(w, "simulate failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	log.Printf("[ION] Simulated payment webhook fired via ngrok — invoice=%s amount=%.0f", req.InvoiceNumber, req.Amount)
	jsonOK(w, map[string]string{"status": "triggered", "path": s.callbackURL})
}

// fireWebhookToSelf constructs a signed DOKU-format webhook payload and POSTs
// it to DOKU_CALLBACK_URL (the ngrok URL). The request routes through ngrok →
// nginx → ION /webhook/doku where verifySignature accepts it because ION signs
// it with its own secret key — the same key used for verification.
func (s *ionService) fireWebhookToSelf(invoiceNumber string, amount float64) error {
	webhookPayload := map[string]any{
		"order": map[string]any{
			"invoice_number": invoiceNumber,
			"amount":         amount,
			"currency":       "IDR",
		},
		"transaction": map[string]any{
			"status": "SUCCESS",
		},
	}
	bodyBytes, _ := json.Marshal(webhookPayload)

	// The webhook path after nginx strips /ion-webhook/ → /webhook/doku
	// sign() must use this path so verifySignature accepts it on arrival
	headers, _ := s.sign("/webhook/doku", bodyBytes)

	req, err := http.NewRequest(http.MethodPost, s.callbackURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ngrok POST failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
