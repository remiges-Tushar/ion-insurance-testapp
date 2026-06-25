package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type DokuService struct {
	clientID  string
	secretKey string
	baseURL   string
}

type DokuVA struct {
	InvoiceNumber string
	RequestID     string
	VANumber      string
	BankCode      string
}

type DokuSplit struct {
	BankAccountSettlementID string
	Value                   float64
	Type                    string // "FIX" or "PERCENTAGE"
}

func NewDokuService() *DokuService {
	baseURL := os.Getenv("DOKU_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api-sandbox.doku.com"
	}
	return &DokuService{
		clientID:  os.Getenv("DOKU_CLIENT_ID"),
		secretKey: os.Getenv("DOKU_SECRET_KEY"),
		baseURL:   baseURL,
	}
}

// newRequestID returns a random UUID v4 string.
func newRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// sign builds DOKU Direct API auth headers for a request.
// Returns the header map and the Request-Id used (needed to store for release).
func (d *DokuService) sign(path string, bodyBytes []byte) (map[string]string, string) {
	requestID := newRequestID()
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	hash := sha256.Sum256(bodyBytes)
	digest := base64.StdEncoding.EncodeToString(hash[:])

	stringToSign := fmt.Sprintf(
		"Client-Id:%s\nRequest-Id:%s\nRequest-Timestamp:%s\nRequest-Target:%s\nDigest:%s",
		d.clientID, requestID, timestamp, path, digest,
	)

	fmt.Printf("[DOKU sign] path=%s requestId=%s\nstringToSign=%q\n", path, requestID, stringToSign)

	mac := hmac.New(sha256.New, []byte(d.secretKey))
	mac.Write([]byte(stringToSign))
	sig := "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"Client-Id":         d.clientID,
		"Request-Id":        requestID,
		"Request-Timestamp": timestamp,
		"Signature":         sig,
	}, requestID
}

func (d *DokuService) doRequest(method, path string, body any) ([]byte, int, string, error) {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, 0, "", err
		}
	}

	headers, requestID := d.sign(path, bodyBytes)

	req, err := http.NewRequest(method, d.baseURL+path, bytes.NewBuffer(bodyBytes))
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
	fmt.Printf("[DOKU response] %s %s → HTTP %d: %s\n", method, path, resp.StatusCode, string(respBytes))
	return respBytes, resp.StatusCode, requestID, err
}

// CreateVirtualAccount creates a DOKU BNI VA with hold_settlement=true (SEAM fund hold).
func (d *DokuService) CreateVirtualAccount(invoiceNumber, customerName string, amount int64) (*DokuVA, error) {
	callbackURL := os.Getenv("DOKU_CALLBACK_URL") // e.g. https://<ngrok>/webhook/doku

	payload := map[string]any{
		"client": map[string]any{"id": d.clientID},
		"order": map[string]any{
			"invoice_number": invoiceNumber,
			"amount":         amount,
			"currency":       "IDR",
			"callback_url":   callbackURL,
		},
		"customer": map[string]any{
			"name": customerName,
		},
		"virtual_account_info": map[string]any{
			"billing_type":    "FIX_BILL",
			"expired_time":    1440, // 24h in minutes
			"reusable_status": false,
		},
		"additional_info": map[string]any{
			"channel":         "VIRTUAL_ACCOUNT_BNI",
			"hold_settlement": true,
		},
	}

	body, status, requestID, err := d.doRequest("POST", "/doku-virtual-account/v2/payment-code", payload)
	if err != nil {
		return nil, fmt.Errorf("doku create VA: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("doku create VA: HTTP %d: %s", status, string(body))
	}

	var resp struct {
		VirtualAccountInfo struct {
			BankCode             string `json:"bank_code"`
			VirtualAccountNumber string `json:"virtual_account_number"`
		} `json:"virtual_account_info"`
		Response struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("doku decode VA response: %w", err)
	}
	if resp.VirtualAccountInfo.VirtualAccountNumber == "" {
		return nil, fmt.Errorf("doku VA: empty VA number (code=%s msg=%s)", resp.Response.Code, resp.Response.Message)
	}

	bankCode := resp.VirtualAccountInfo.BankCode
	if bankCode == "" {
		bankCode = "BNI"
	}
	return &DokuVA{
		InvoiceNumber: invoiceNumber,
		RequestID:     requestID,
		VANumber:      resp.VirtualAccountInfo.VirtualAccountNumber,
		BankCode:      bankCode,
	}, nil
}

// ReleaseSettlement releases held funds, optionally splitting to multiple accounts.
// If splits is empty, DOKU releases to the default settlement account.
func (d *DokuService) ReleaseSettlement(invoiceNumber, originalRequestID string, amount int64, splits []DokuSplit) error {
	overrides := make([]map[string]any, 0, len(splits))
	for _, s := range splits {
		overrides = append(overrides, map[string]any{
			"bank_account_settlement_id": s.BankAccountSettlementID,
			"value":                      s.Value,
			"type":                       s.Type,
		})
	}

	payload := map[string]any{
		"order": map[string]any{
			"invoice_number": invoiceNumber,
			"amount":         amount,
			"currency":       "IDR",
		},
		"transaction": map[string]any{
			"original_request_id": originalRequestID,
		},
		"override_settlement": overrides,
	}

	body, status, _, err := d.doRequest("POST", "/finance/v1/release", payload)
	if err != nil {
		return fmt.Errorf("doku release: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("doku release: HTTP %d: %s", status, string(body))
	}
	return nil
}

// VerifyWebhookSignature validates the HMACSHA256 signature on a DOKU webhook.
// requestTarget is the path the webhook was delivered to, e.g. "/webhook/doku".
func (d *DokuService) VerifyWebhookSignature(clientID, requestID, timestamp, requestTarget string, bodyBytes []byte, signature string) bool {
	if d.secretKey == "" {
		return true // bypass when unconfigured (dev/test)
	}

	hash := sha256.Sum256(bodyBytes)
	digest := base64.StdEncoding.EncodeToString(hash[:])

	stringToSign := fmt.Sprintf(
		"Client-Id:%s\nRequest-Id:%s\nRequest-Timestamp:%s\nRequest-Target:%s\nDigest:%s",
		clientID, requestID, timestamp, requestTarget, digest,
	)

	mac := hmac.New(sha256.New, []byte(d.secretKey))
	mac.Write([]byte(stringToSign))
	expected := "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}
