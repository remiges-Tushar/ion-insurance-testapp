package service

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type XenditService struct {
	secretKey    string
	webhookToken string
	baseURL      string
}

type XenditVA struct {
	ID             string `json:"id"`
	ExternalID     string `json:"external_id"`
	BankCode       string `json:"bank_code"`
	AccountNumber  string `json:"account_number"`
	Status         string `json:"status"`
	ExpectedAmount int64  `json:"expected_amount"`
}

type XenditVAPayment struct {
	ID                       string  `json:"id"`
	CallbackVirtualAccountID string  `json:"callback_virtual_account_id"`
	ExternalID               string  `json:"external_id"`
	Amount                   float64 `json:"amount"`
	BankCode                 string  `json:"bank_code"`
	AccountNumber            string  `json:"account_number"`
	TransactionTimestamp     string  `json:"transaction_timestamp"`
}

type xenditVAPaymentsResponse struct {
	Data []XenditVAPayment `json:"data"`
}

func NewXenditService() *XenditService {
	baseURL := os.Getenv("XENDIT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.xendit.co"
	}
	return &XenditService{
		secretKey:    os.Getenv("XENDIT_SECRET_KEY"),
		webhookToken: os.Getenv("XENDIT_WEBHOOK_TOKEN"),
		baseURL:      baseURL,
	}
}

func (x *XenditService) doRequest(method, path string, body any, extraHeaders ...string) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, x.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(x.secretKey, "")
	req.Header.Set("Content-Type", "application/json")
	// extraHeaders: key, value, key, value, ...
	for i := 0; i+1 < len(extraHeaders); i += 2 {
		req.Header.Set(extraHeaders[i], extraHeaders[i+1])
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, err
}

func (x *XenditService) CreateVirtualAccount(externalID, name string, amount int64) (*XenditVA, error) {
	payload := map[string]any{
		"external_id":     externalID,
		"bank_code":       "BNI",
		"name":            name,
		"expected_amount": amount,
		"is_single_use":   true,
		"is_closed":       true,
		"expiration_date": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}

	body, status, err := x.doRequest("POST", "/callback_virtual_accounts", payload)
	if err != nil {
		return nil, fmt.Errorf("xendit create VA: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("xendit create VA: HTTP %d: %s", status, string(body))
	}

	var va XenditVA
	if err := json.Unmarshal(body, &va); err != nil {
		return nil, fmt.Errorf("xendit decode VA: %w", err)
	}
	if va.ID == "" {
		return nil, fmt.Errorf("xendit VA creation returned empty ID")
	}
	return &va, nil
}

// SimulateVAPayment triggers a sandbox payment on a VA (sandbox only).
func (x *XenditService) SimulateVAPayment(externalID string, amount int64) error {
	payload := map[string]any{"amount": amount}
	body, status, err := x.doRequest("POST",
		"/callback_virtual_accounts/external_id="+externalID+"/simulate_payment", payload)
	if err != nil {
		return fmt.Errorf("xendit simulate VA: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("xendit simulate VA: HTTP %d: %s", status, string(body))
	}
	return nil
}

// SimulateQRISPayment triggers a sandbox payment on a QRIS code (sandbox only).
func (x *XenditService) SimulateQRISPayment(qrisID string, amount int64) error {
	payload := map[string]any{"amount": amount}
	body, status, err := x.doRequest("POST",
		"/qr_codes/"+qrisID+"/payments/simulate", payload, "api-version", "2022-07-31")
	if err != nil {
		return fmt.Errorf("xendit simulate QRIS: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("xendit simulate QRIS: HTTP %d: %s", status, string(body))
	}
	return nil
}

func (x *XenditService) GetVAPayments(xenditVAID string) ([]XenditVAPayment, error) {
	body, status, err := x.doRequest("GET", "/callback_virtual_accounts/"+xenditVAID+"/payments", nil)
	if err != nil {
		return nil, fmt.Errorf("xendit get VA payments: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("xendit get VA payments: HTTP %d: %s", status, string(body))
	}

	var result xenditVAPaymentsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("xendit decode VA payments: %w", err)
	}
	return result.Data, nil
}

func (x *XenditService) VerifyToken(token string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(x.webhookToken)) == 1
}

// XenditQRIS holds the response from a QRIS creation call.
type XenditQRIS struct {
	ID          string `json:"id"`
	ReferenceID string `json:"reference_id"`
	QRString    string `json:"qr_string"`
	Amount      int64  `json:"amount"`
	Status      string `json:"status"`
}

// XenditQRISPayment holds one QRIS payment record.
type XenditQRISPayment struct {
	ID          string  `json:"id"`
	QRID        string  `json:"qr_id"`
	ReferenceID string  `json:"reference_id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
}

type xenditQRISPaymentsResponse struct {
	Data []XenditQRISPayment `json:"data"`
}

// CreateQRIS creates a dynamic QRIS code for the given amount.
func (x *XenditService) CreateQRIS(referenceID string, amount int64) (*XenditQRIS, error) {
	payload := map[string]any{
		"reference_id": referenceID,
		"type":         "DYNAMIC",
		"currency":     "IDR",
		"amount":       amount,
		"expires_at":   time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}

	body, status, err := x.doRequest("POST", "/qr_codes", payload, "api-version", "2022-07-31")
	if err != nil {
		return nil, fmt.Errorf("xendit create QRIS: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("xendit create QRIS: HTTP %d: %s", status, string(body))
	}

	var qr XenditQRIS
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, fmt.Errorf("xendit decode QRIS: %w", err)
	}
	if qr.ID == "" {
		return nil, fmt.Errorf("xendit QRIS creation returned empty ID")
	}
	return &qr, nil
}

// GetQRISPayments returns payments made against a QRIS code.
func (x *XenditService) GetQRISPayments(qrisID string) ([]XenditQRISPayment, error) {
	body, status, err := x.doRequest("GET", "/qr_codes/"+qrisID+"/payments", nil, "api-version", "2022-07-31")
	if err != nil {
		return nil, fmt.Errorf("xendit get QRIS payments: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("xendit get QRIS payments: HTTP %d: %s", status, string(body))
	}

	var result xenditQRISPaymentsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("xendit decode QRIS payments: %w", err)
	}
	return result.Data, nil
}
