package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ionSchemaBaseBAP = "https://raw.githubusercontent.com/remiges-Tushar/ion-specs/refs/heads/feat/FIN-03-motor-insurance-schema/schema/extensions/finance/"

// ClientService is the core business-logic layer for the BAP application.
// It mediates between the frontend-facing HTTP handlers and the Beckn network
// via onix-bap, using Redis pub/sub to turn async Beckn callbacks into
// synchronous frontend responses.
type ClientService struct {
	db               *pgxpool.Pool
	cb               *CallbackManager
	onixBAPCallerURL string
	ionServiceURL    string
	bppWebhookURL    string
	bapFrontendURL   string
}

// NewClientService constructs a ClientService.
// onixBAPCallerURL is typically read from env BAP_ONIX_BAP_CALLER_URL.
// ionServiceURL is typically read from env ION_SERVICE_URL.
func NewClientService(db *pgxpool.Pool, cb *CallbackManager, onixBAPCallerURL, ionServiceURL string) *ClientService {
	bppURL := os.Getenv("BPP_WEBHOOK_URL")
	if bppURL == "" {
		bppURL = "http://bpp:8080/webhook"
	}
	bapFrontendURL := os.Getenv("BAP_FRONTEND_URL")
	if bapFrontendURL == "" {
		bapFrontendURL = "http://localhost:3000"
	}
	return &ClientService{
		db:               db,
		cb:               cb,
		onixBAPCallerURL: onixBAPCallerURL,
		ionServiceURL:    ionServiceURL,
		bppWebhookURL:    bppURL,
		bapFrontendURL:   bapFrontendURL,
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildContext creates a minimal Beckn context map.
func (s *ClientService) buildContext(action, txnId, msgId string) map[string]any {
	return map[string]any{
		"version":        "2.0.0",
		"action":         action,
		"domain":         "ion:finance",
		"bap_id":         "insurance-bap.iontest",
		"bap_uri":        "http://onix-bap:8081/bap/receiver",
		"transaction_id": txnId,
		"message_id":     msgId,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"ttl":            "PT30S",
	}
}

// forwardAndWait is the central async→sync bridge:
//  1. Extract txnId + msgId from payload["context"].
//  2. Register a pending slot in Redis.
//  3. Forward the payload to onix-bap (fire & forget — onix-bap returns ACK).
//  4. Block on Wait until the on_* webhook arrives and calls Publish.
func (s *ClientService) forwardAndWait(ctx context.Context, action string, payload map[string]any) ([]byte, error) {
	ctxMap, _ := payload["context"].(map[string]any)
	txnId, _ := ctxMap["transaction_id"].(string)
	msgId, _ := ctxMap["message_id"].(string)

	if err := s.cb.Register(ctx, action, txnId, msgId); err != nil {
		return nil, fmt.Errorf("register callback: %w", err)
	}

	if err := s.callOnixCaller(action, payload); err != nil {
		return nil, fmt.Errorf("call onix-bap/%s: %w", action, err)
	}

	body, err := s.cb.Wait(ctx, action, txnId, msgId, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("wait for on_%s: %w", action, err)
	}
	return body, nil
}

// callOnixCaller POSTs the payload to onixBAPCallerURL/<action>.
func (s *ClientService) callOnixCaller(action string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := s.onixBAPCallerURL + "/" + action
	resp, err := http.Post(url, "application/json", bytes.NewReader(data)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onix-bap returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// upsertTransaction inserts or updates a transaction row.
func (s *ClientService) upsertTransaction(ctx context.Context, txnId, action, status string) {
	_, err := s.db.Exec(ctx,
		`INSERT INTO transactions (transaction_id, action, status)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (transaction_id) DO UPDATE SET status = EXCLUDED.status`,
		txnId, action, status,
	)
	if err != nil {
		log.Printf("[ClientService] upsertTransaction error: %v", err)
	}
}

// saveSnapshot inserts a callback snapshot into contract_snapshots.
func (s *ClientService) saveSnapshot(ctx context.Context, txnId, onAction string, payload []byte) {
	_, err := s.db.Exec(ctx,
		`INSERT INTO contract_snapshots (transaction_id, on_action, payload)
		 VALUES ($1, $2, $3)`,
		txnId, onAction, payload,
	)
	if err != nil {
		log.Printf("[ClientService] saveSnapshot error: %v", err)
	}
}

// logMessage inserts a raw message into beckn_message_log.
func (s *ClientService) logMessage(ctx context.Context, action, txnId string, payload map[string]any) {
	data, _ := json.Marshal(payload)
	_, err := s.db.Exec(ctx,
		`INSERT INTO beckn_message_log (action, transaction_id, payload)
		 VALUES ($1, $2, $3)`,
		action, txnId, data,
	)
	if err != nil {
		log.Printf("[ClientService] logMessage error: %v", err)
	}
}

// newTxnMsg generates a fresh transaction_id and message_id pair.
func newTxnMsg() (txnId, msgId string) {
	return uuid.NewString(), uuid.NewString()
}

// parseResponse unmarshals raw bytes into map[string]any.
func parseResponse(body []byte) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Frontend-facing business methods
// ---------------------------------------------------------------------------

// Discover performs a catalogue discovery via the full Beckn discover flow.
func (s *ClientService) Discover(ctx context.Context, query string) (map[string]any, error) {
	return s.Search(ctx, query)
}

// Search sends a Beckn v2 discover message to the network and waits for on_discover.
func (s *ClientService) Search(ctx context.Context, query string) (map[string]any, error) {
	txnId, msgId := newTxnMsg()
	payload := map[string]any{
		"context": s.buildContext("discover", txnId, msgId),
		"message": map[string]any{
			"intent": map[string]any{
				"textSearch": query,
			},
		},
	}
	s.logMessage(ctx, "discover", txnId, payload)
	s.upsertTransaction(ctx, txnId, "discover", "INITIATED")

	body, err := s.forwardAndWait(ctx, "discover", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// Select sends a select message and waits for on_select (QUOTE_RECEIVED).
// req is the UI vehicle-details form: {idv, productId, productType, tariffZone, ...}.
func (s *ClientService) Select(ctx context.Context, req map[string]any) (map[string]any, error) {
	txnId, msgId := s.extractOrNew(req)

	productId := stringOrDefault(req["productId"], "1")
	productType := stringOrDefault(req["productType"], "MOTOR_COMPREHENSIVE")
	tariffZone := stringOrDefault(req["tariffZone"], "ZONE_3")
	var idv float64
	if v, ok := req["idv"].(float64); ok {
		idv = v
	}

	payload := map[string]any{
		"context": s.buildContext("select", txnId, msgId),
		"message": map[string]any{
			"contract": map[string]any{
				"commitments": []any{
					map[string]any{
						"status":    map[string]any{"descriptor": map[string]any{"code": "DRAFT"}},
						"resources": []any{map[string]any{"id": productId, "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{productId}},
						"commitmentAttributes": map[string]any{
							"@context":    ionSchemaBaseBAP + "insurance-contract/v1/context.jsonld",
							"@type":       "InsurancePolicy",
							"productType": productType,
							"tariffZone":  tariffZone,
							"idv":         idv,
						},
					},
				},
			},
		},
	}
	s.logMessage(ctx, "select", txnId, payload)
	s.upsertTransaction(ctx, txnId, "select", "SELECT_SENT")

	body, err := s.forwardAndWait(ctx, "select", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// Init sends an init message and waits for on_init (INIT_RECEIVED).
// req is the UI KYC form: {nik, name, dob, transaction_id, idv, ...}.
func (s *ClientService) Init(ctx context.Context, req map[string]any) (map[string]any, error) {
	txnId, msgId := s.extractOrNew(req)

	nik := stringOrDefault(req["nik"], "")
	var idv float64
	if v, ok := req["idv"].(float64); ok {
		idv = v
	}
	vin := stringOrDefault(req["plate"], stringOrDefault(req["vin"], "VIN-UNKNOWN"))

	payload := map[string]any{
		"context": s.buildContext("init", txnId, msgId),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnId,
				"participants": []any{
					map[string]any{
						"id": nik,
						"participantAttributes": map[string]any{
							"@context": ionSchemaBaseBAP + "insurance-participant/v1/context.jsonld",
							"@type":    "Policyholder",
							"nik":      nik,
						},
					},
				},
				"commitments": []any{
					map[string]any{
						"status":    map[string]any{"descriptor": map[string]any{"code": "DRAFT"}},
						"resources": []any{map[string]any{"id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context": ionSchemaBaseBAP + "insurance-contract/v1/context.jsonld",
							"@type":    "InsurancePolicy",
							"vin":      vin,
							"idv":      idv,
						},
					},
				},
			},
		},
	}
	s.logMessage(ctx, "init", txnId, payload)
	s.upsertTransaction(ctx, txnId, "init", "INIT_SENT")

	body, err := s.forwardAndWait(ctx, "init", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// Confirm sends a confirm message and waits for on_confirm (CONFIRMED).
// req is the UI payment form: {transaction_id, paymentRef, amount, paymentMethod}.
func (s *ClientService) Confirm(ctx context.Context, req map[string]any) (map[string]any, error) {
	txnId, msgId := s.extractOrNew(req)

	paymentMethod := stringOrDefault(req["paymentMethod"], "VIRTUAL_ACCOUNT")
	paymentRef := stringOrDefault(req["paymentRef"], fmt.Sprintf("VA-PAY-%s", txnId[:8]))
	var amount float64
	if v, ok := req["amount"].(float64); ok {
		amount = v
	}

	payload := map[string]any{
		"context": s.buildContext("confirm", txnId, msgId),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnId,
				"commitments": []any{
					map[string]any{
						"status":    map[string]any{"descriptor": map[string]any{"code": "ACTIVE"}},
						"resources": []any{map[string]any{"id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context": ionSchemaBaseBAP + "insurance-contract/v1/context.jsonld",
							"@type":    "InsurancePolicy",
							"payment": map[string]any{
								"method": paymentMethod,
								"ref":    paymentRef,
								"amount": amount,
							},
						},
					},
				},
			},
		},
	}
	s.logMessage(ctx, "confirm", txnId, payload)
	s.upsertTransaction(ctx, txnId, "confirm", "CONFIRM_SENT")

	body, err := s.forwardAndWait(ctx, "confirm", payload)
	if err != nil {
		return nil, err
	}
	result, err := parseResponse(body)
	if err != nil {
		return nil, err
	}
	if errField, ok := result["error"].(map[string]any); ok {
		msg, _ := errField["message"].(string)
		if msg == "" {
			msg = "confirm rejected by BPP"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return result, nil
}

// RequestStatus sends a status request to the network and waits for on_status.
func (s *ClientService) RequestStatus(ctx context.Context, txnId string) (map[string]any, error) {
	msgId := uuid.NewString()
	payload := map[string]any{
		"context": s.buildContext("status", txnId, msgId),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnId,
				"commitments": []any{
					map[string]any{
						"status":    map[string]any{"descriptor": map[string]any{"code": "ACTIVE"}},
						"resources": []any{map[string]any{"id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{"1"}},
					},
				},
			},
		},
	}
	s.logMessage(ctx, "status", txnId, payload)

	body, err := s.forwardAndWait(ctx, "status", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// Cancel sends a cancel request and waits for on_cancel (CANCELLED).
func (s *ClientService) Cancel(ctx context.Context, txnId string) (map[string]any, error) {
	msgId := uuid.NewString()
	payload := map[string]any{
		"context": s.buildContext("cancel", txnId, msgId),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnId,
				"commitments": []any{
					map[string]any{
						"status":    map[string]any{"descriptor": map[string]any{"code": "CLOSED"}},
						"resources": []any{map[string]any{"id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{"1"}},
					},
				},
			},
		},
	}
	s.logMessage(ctx, "cancel", txnId, payload)
	s.upsertTransaction(ctx, txnId, "cancel", "CANCEL_SENT")

	body, err := s.forwardAndWait(ctx, "cancel", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// Rate sends a rating and waits for on_rate.
func (s *ClientService) Rate(ctx context.Context, req map[string]any) (map[string]any, error) {
	txnId, msgId := s.extractOrNew(req)

	ratingInput := map[string]any{
		"target": map[string]any{"id": txnId},
		"range":  map[string]any{},
	}
	if score, ok := req["score"].(float64); ok && score > 0 {
		ratingInput["range"] = map[string]any{"value": score, "min": 1.0, "max": 5.0}
	}
	if feedback, ok := req["feedback"].(string); ok && feedback != "" {
		ratingInput["feedbackFormSubmission"] = map[string]any{
			"data": map[string]any{"feedback": feedback},
		}
	}

	payload := map[string]any{
		"context": s.buildContext("rate", txnId, msgId),
		"message": map[string]any{
			"ratingInputs": []any{ratingInput},
		},
	}
	s.logMessage(ctx, "rate", txnId, payload)

	body, err := s.forwardAndWait(ctx, "rate", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// Support sends a support request and waits for on_support.
func (s *ClientService) Support(ctx context.Context, req map[string]any) (map[string]any, error) {
	txnId, msgId := s.extractOrNew(req)

	description, _ := req["description"].(string)
	if description == "" {
		description = "Support request"
	}

	payload := map[string]any{
		"context": s.buildContext("support", txnId, msgId),
		"message": map[string]any{
			"support": map[string]any{
				"orderId": txnId,
				"descriptor": map[string]any{
					"name": description,
				},
				"channels": []any{
					map[string]any{"type": "EMAIL"},
				},
			},
		},
	}
	s.logMessage(ctx, "support", txnId, payload)

	body, err := s.forwardAndWait(ctx, "support", payload)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// GetStatus does a non-blocking DB lookup: returns the transaction status plus
// the most recent contract snapshot payload.
func (s *ClientService) GetStatus(ctx context.Context, txnId string) (map[string]any, error) {
	var action, status string
	err := s.db.QueryRow(ctx,
		`SELECT action, status FROM transactions WHERE transaction_id = $1`,
		txnId,
	).Scan(&action, &status)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	var snapshotPayload []byte
	var onAction string
	_ = s.db.QueryRow(ctx,
		`SELECT on_action, payload FROM contract_snapshots
		 WHERE transaction_id = $1
		 ORDER BY received_at DESC LIMIT 1`,
		txnId,
	).Scan(&onAction, &snapshotPayload)

	result := map[string]any{
		"transaction_id": txnId,
		"action":         action,
		"status":         status,
	}
	if len(snapshotPayload) > 0 {
		var snap map[string]any
		if err := json.Unmarshal(snapshotPayload, &snap); err == nil {
			result["latest_snapshot"] = snap
			result["latest_on_action"] = onAction
		}
	}
	return result, nil
}

// HandleCallback is called by the on_* webhook handlers.  It saves the
// snapshot, updates the transaction status, and publishes to Redis to unblock
// the waiting frontend handler.
func (s *ClientService) HandleCallback(ctx context.Context, onAction string, payload map[string]any) error {
	ctxMap, _ := payload["context"].(map[string]any)
	txnId, _ := ctxMap["transaction_id"].(string)
	msgId, _ := ctxMap["message_id"].(string)

	// SEAM Stage 1: on on_init, BAP calls ION to create VA and QRIS, then injects
	// the payment instrument details into the payload before saving snapshot + publish.
	// This enriched payload is what the frontend sees in the response to Init().
	if onAction == "on_init" && txnId != "" {
		// Upsert first so the UPDATE for doku_va_number has a row to hit.
		s.upsertTransaction(ctx, txnId, "init", "INIT_RECEIVED")
		totalIDR := extractTotalIDRFromOnInit(payload)
		invoiceNumber := "ins-" + txnId

		ionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if co, err := s.callIONCreateCheckout(ionCtx, invoiceNumber, totalIDR, txnId); err == nil {
			injectIntoConsiderationAttributes(payload, map[string]any{
				"checkoutUrl":   co["checkout_url"],
				"invoiceNumber": co["invoice_number"],
			})
			s.db.Exec(ctx,
				`UPDATE transactions SET doku_invoice_number=$1, doku_va_number=$2, payment_amount=$3 WHERE transaction_id=$4`,
				co["invoice_number"], co["checkout_url"], totalIDR, txnId)
			log.Printf("[SEAM Stage 1] Checkout created via ION — invoice=%s url=%s", co["invoice_number"], co["checkout_url"])
		} else {
			log.Printf("[BAP] ION create-checkout failed: %v", err)
		}
	}

	// Persist snapshot and audit log (with enriched payload).
	data, _ := json.Marshal(payload)
	if txnId != "" {
		s.saveSnapshot(ctx, txnId, onAction, data)
		s.logMessage(ctx, onAction, txnId, payload)
	}

	// Derive forward route from onAction (strip "on_" prefix).
	route := strings.TrimPrefix(onAction, "on_")

	// Update transaction status.
	switch onAction {
	case "on_select":
		s.upsertTransaction(ctx, txnId, "select", "QUOTE_RECEIVED")
	case "on_init":
		s.upsertTransaction(ctx, txnId, "init", "INIT_RECEIVED")
	case "on_confirm":
		if _, hasError := payload["error"]; hasError {
			log.Printf("[ClientService] on_confirm error for txn=%s — not marking CONFIRMED", txnId)
		} else {
			s.upsertTransaction(ctx, txnId, "confirm", "CONFIRMED")
			policyState := extractPolicyStateFromOnConfirm(payload)
			if policyState == "ACTIVE" {
				go s.sendReconcile(context.Background(), txnId)
			}
		}
	case "on_reconcile":
		s.upsertTransaction(ctx, txnId, "reconcile", "SETTLED")
	case "on_cancel":
		s.upsertTransaction(ctx, txnId, "cancel", "CANCELLED")
	// on_search, on_status, on_rate, on_support, on_reconcile — publish may fail gracefully
	}

	// Publish to unblock the waiting forwardAndWait call.
	// on_reconcile has no waiting forwardAndWait (fire-and-forget), so publish error is expected.
	if err := s.cb.Publish(ctx, route, txnId, msgId, data); err != nil {
		log.Printf("[ClientService] Publish for onAction=%s txnId=%s: %v", onAction, txnId, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ION service helpers
// ---------------------------------------------------------------------------

// NotifyPaymentReceived tells ION that the customer paid, bypassing the external DOKU webhook.
// ION then notifies BPP directly to set payment_received = true.
func (s *ClientService) NotifyPaymentReceived(ctx context.Context, txnId string) error {
	var invoiceNumber string
	var amount int64
	var checkoutReqID string
	s.db.QueryRow(ctx,
		`SELECT COALESCE(doku_invoice_number,''), COALESCE(payment_amount,0),
		        COALESCE(doku_checkout_request_id,'')
		 FROM transactions WHERE transaction_id=$1`,
		txnId,
	).Scan(&invoiceNumber, &amount, &checkoutReqID)
	if invoiceNumber == "" {
		invoiceNumber = "ins-" + txnId
	}
	body, _ := json.Marshal(map[string]any{
		"invoice_number":     invoiceNumber,
		"amount":             float64(amount),
		"payment_request_id": checkoutReqID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.ionServiceURL+"/payment/notify-direct", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notify-direct HTTP %d: %s", resp.StatusCode, string(b))
	}
	log.Printf("[BAP] Payment notified via ION direct — txn=%s invoice=%s", txnId, invoiceNumber)
	return nil
}

// callIONCreateCheckout calls the ION service to create a DOKU Checkout session.
// txnID is used to build a deep-link callback_url so DOKU redirects back to the
// exact payment step in the BAP frontend after the customer pays.
// Returns map with "checkout_url" and "invoice_number".
func (s *ClientService) callIONCreateCheckout(ctx context.Context, invoiceNumber string, amountIDR int64, txnID string) (map[string]any, error) {
	if s.ionServiceURL == "" {
		return nil, fmt.Errorf("ION_SERVICE_URL not configured")
	}
	callbackURL := s.bapFrontendURL + "/policy/" + txnID + "?payment=done"
	body, _ := json.Marshal(map[string]any{
		"invoice_number": invoiceNumber,
		"customer_name":  "ION Insurance Customer",
		"amount_idr":     amountIDR,
		"callback_url":   callbackURL,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ionServiceURL+"/payment/create-checkout", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ION create-checkout HTTP %d: %s", resp.StatusCode, string(respBytes))
	}
	var result map[string]any
	json.Unmarshal(respBytes, &result)

	// Store the checkout Request-Id so NotifyPaymentReceived can forward it to BPP.
	// BPP needs it as doku_request_id for the DOKU settlement release API.
	if rid, _ := result["payment_request_id"].(string); rid != "" {
		s.db.Exec(ctx,
			`UPDATE transactions SET doku_checkout_request_id=$2 WHERE transaction_id=$1`,
			txnID, rid,
		)
	}

	return result, nil
}

// callIONCreateVA calls the ION service to create a DOKU VA for the given invoice.
func (s *ClientService) callIONCreateVA(ctx context.Context, invoiceNumber string, amountIDR int64) (map[string]any, error) {
	if s.ionServiceURL == "" {
		return nil, fmt.Errorf("ION_SERVICE_URL not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"invoice_number": invoiceNumber,
		"customer_name":  "ION Insurance",
		"amount_idr":     amountIDR,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ionServiceURL+"/payment/create-va", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ION create-va HTTP %d: %s", resp.StatusCode, string(respBytes))
	}
	var result map[string]any
	json.Unmarshal(respBytes, &result)
	return result, nil
}

// callIONCreateQRIS calls the ION service to create a DOKU QRIS code.
// Returns the QR string content.
func (s *ClientService) callIONCreateQRIS(ctx context.Context, invoiceNumber string, amountIDR int64) (string, error) {
	if s.ionServiceURL == "" {
		return "", fmt.Errorf("ION_SERVICE_URL not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"invoice_number": invoiceNumber,
		"customer_name":  "ION Insurance",
		"amount_idr":     amountIDR,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ionServiceURL+"/payment/create-qris", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ION create-qris HTTP %d: %s", resp.StatusCode, string(respBytes))
	}
	var result map[string]any
	json.Unmarshal(respBytes, &result)
	qr, _ := result["qr_string"].(string)
	return qr, nil
}

// sendReconcile auto-sends a Beckn reconcile message to BPP after on_confirm.
// Called asynchronously (go s.sendReconcile).
func (s *ClientService) sendReconcile(ctx context.Context, txnId string) {
	var invoiceNumber string
	var amount int64
	s.db.QueryRow(ctx,
		`SELECT COALESCE(doku_invoice_number,''), COALESCE(payment_amount,0) FROM transactions WHERE transaction_id=$1`,
		txnId,
	).Scan(&invoiceNumber, &amount)

	if invoiceNumber == "" {
		invoiceNumber = "ins-" + txnId
	}
	if amount == 0 {
		amount = s.getPaymentAmountFromSnapshot(ctx, txnId)
	}

	msgId := uuid.NewString()
	payload := map[string]any{
		"context": s.buildContext("reconcile", txnId, msgId),
		"message": map[string]any{
			"settlement": map[string]any{
				"invoice_number": invoiceNumber,
				"total_amount":   amount,
				"splits": map[string]any{
					"seller_pct":    97,
					"buyer_app_pct": 3,
				},
			},
		},
	}
	s.upsertTransaction(ctx, txnId, "reconcile", "RECONCILING")
	// Onix does not support SEAM 'reconcile' — send directly to BPP webhook
	if err := s.postDirect(s.bppWebhookURL+"/reconcile", payload); err != nil {
		log.Printf("[BAP] sendReconcile failed: %v", err)
	} else {
		log.Printf("[SEAM Stage 4] Reconcile sent directly to BPP — invoice=%s amount=%d", invoiceNumber, amount)
	}
}

func (s *ClientService) postDirect(url string, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s HTTP %d: %s", url, resp.StatusCode, string(respBody))
	}
	return nil
}

// getPaymentAmountFromSnapshot extracts totalAmountIDR from the on_init contract_snapshot.
func (s *ClientService) getPaymentAmountFromSnapshot(ctx context.Context, txnId string) int64 {
	var snapshotData []byte
	s.db.QueryRow(ctx,
		`SELECT payload FROM contract_snapshots WHERE transaction_id=$1 AND on_action='on_init' ORDER BY received_at DESC LIMIT 1`,
		txnId,
	).Scan(&snapshotData)
	if len(snapshotData) == 0 {
		return 500_000
	}
	var snap map[string]any
	json.Unmarshal(snapshotData, &snap)
	return extractTotalIDRFromOnInit(snap)
}

// ---------------------------------------------------------------------------
// Payload mutation helpers (for SEAM VA injection)
// ---------------------------------------------------------------------------

// extractTotalIDRFromOnInit reads totalAmountIDR from the on_init payload's considerationAttributes.
func extractTotalIDRFromOnInit(payload map[string]any) int64 {
	msg, _ := payload["message"].(map[string]any)
	contract, _ := msg["contract"].(map[string]any)
	commitments, _ := contract["commitments"].([]any)
	if len(commitments) == 0 {
		return 500_000
	}
	c, _ := commitments[0].(map[string]any)
	ca, _ := c["considerationAttributes"].(map[string]any)
	if total, ok := ca["totalAmountIDR"].(float64); ok {
		return int64(total)
	}
	return 500_000
}

// injectIntoConsiderationAttributes merges extra fields into commitments[0].considerationAttributes.
func injectIntoConsiderationAttributes(payload map[string]any, extra map[string]any) {
	msg, _ := payload["message"].(map[string]any)
	contract, _ := msg["contract"].(map[string]any)
	commitments, _ := contract["commitments"].([]any)
	if len(commitments) == 0 {
		return
	}
	c, _ := commitments[0].(map[string]any)
	ca, _ := c["considerationAttributes"].(map[string]any)
	if ca == nil {
		return
	}
	for k, v := range extra {
		ca[k] = v
	}
}

// extractPolicyStateFromOnConfirm reads policyState from the on_confirm payload's commitmentAttributes.
func extractPolicyStateFromOnConfirm(payload map[string]any) string {
	msg, _ := payload["message"].(map[string]any)
	contract, _ := msg["contract"].(map[string]any)
	commitments, _ := contract["commitments"].([]any)
	if len(commitments) == 0 {
		return ""
	}
	c, _ := commitments[0].(map[string]any)
	ca, _ := c["commitmentAttributes"].(map[string]any)
	state, _ := ca["policyState"].(string)
	return state
}

// ListPolicies returns enriched confirmed policy data by joining on_confirm and on_select snapshots.
func (s *ClientService) ListPolicies(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT cs.transaction_id, cs.payload, cs.received_at,
		        COALESCE(cs2.payload, '{}'::jsonb) as select_payload
		 FROM contract_snapshots cs
		 LEFT JOIN contract_snapshots cs2
		   ON cs2.transaction_id = cs.transaction_id AND cs2.on_action = 'on_select'
		 WHERE cs.on_action = 'on_confirm'
		 ORDER BY cs.received_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []map[string]any
	for rows.Next() {
		var txnId string
		var confirmPayload, selectPayload []byte
		var receivedAt time.Time
		if err := rows.Scan(&txnId, &confirmPayload, &receivedAt, &selectPayload); err != nil {
			continue
		}
		var confirm, sel map[string]any
		json.Unmarshal(confirmPayload, &confirm)
		json.Unmarshal(selectPayload, &sel)

		// Extract from on_confirm: commitmentAttributes
		ca := nestedMap(confirm, "message", "contract", "commitments", 0, "commitmentAttributes")
		// Extract from on_select: offerAttributes
		oa := nestedMap(sel, "message", "contract", "commitments", 0, "offer", "offerAttributes")

		policy := map[string]any{
			"transaction_id":  txnId,
			"received_at":     receivedAt,
			"policy_number":   stringField(ca, "policyNumber"),
			"certificate_url": stringField(ca, "certificateUrl"),
			"coverage_start":  stringField(ca, "coverageStart"),
			"coverage_end":    stringField(ca, "coverageEnd"),
			"status":          stringField(ca, "policyStatus"),
			"annual_premium":  numField(oa, "annualPremiumIDR"),
			"idv":             numField(oa, "approvedIDV"),
		}
		if policy["status"] == "" {
			policy["status"] = "ACTIVE"
		}
		policies = append(policies, policy)
	}
	if policies == nil {
		policies = []map[string]any{}
	}
	return policies, nil
}

// nestedMap navigates a nested map/slice path; keys can be string or int index.
func nestedMap(m map[string]any, keys ...any) map[string]any {
	var cur any = m
	for _, k := range keys {
		if cur == nil {
			return nil
		}
		switch kv := k.(type) {
		case string:
			mm, ok := cur.(map[string]any)
			if !ok {
				return nil
			}
			cur = mm[kv]
		case int:
			sl, ok := cur.([]any)
			if !ok || kv >= len(sl) {
				return nil
			}
			cur = sl[kv]
		}
	}
	res, _ := cur.(map[string]any)
	return res
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func numField(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	}
	return 0
}

// GetPolicyByTxn returns the latest on_confirm snapshot for a given txnId.
func (s *ClientService) GetPolicyByTxn(ctx context.Context, txnId string) (map[string]any, error) {
	var payload []byte
	var receivedAt time.Time
	err := s.db.QueryRow(ctx,
		`SELECT payload, received_at FROM contract_snapshots
		 WHERE transaction_id = $1 AND on_action = 'on_confirm'
		 ORDER BY received_at DESC LIMIT 1`,
		txnId,
	).Scan(&payload, &receivedAt)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}
	var snap map[string]any
	if err := json.Unmarshal(payload, &snap); err != nil {
		snap = map[string]any{}
	}
	return map[string]any{
		"transaction_id": txnId,
		"received_at":    receivedAt,
		"snapshot":       snap,
	}, nil
}

// ListOrders returns all transactions with SEAM stage info, joining the latest snapshot.
func (s *ClientService) ListOrders(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT t.transaction_id, t.action, t.status, t.created_at,
		        COALESCE(t.doku_invoice_number,''), COALESCE(t.doku_va_number,''),
		        COALESCE(t.payment_amount,0),
		        COALESCE(cs.on_action,''), COALESCE(cs.received_at, t.created_at)
		 FROM transactions t
		 LEFT JOIN LATERAL (
		     SELECT on_action, received_at FROM contract_snapshots
		     WHERE transaction_id = t.transaction_id
		     ORDER BY received_at DESC LIMIT 1
		 ) cs ON true
		 ORDER BY t.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []map[string]any
	for rows.Next() {
		var txnId, action, status, invoiceNum, vaNum, latestOnAction string
		var createdAt, latestAt time.Time
		var amount int64
		if err := rows.Scan(&txnId, &action, &status, &createdAt,
			&invoiceNum, &vaNum, &amount, &latestOnAction, &latestAt); err != nil {
			continue
		}
		seamStage := seamStageFromStatus(status, latestOnAction)
		orders = append(orders, map[string]any{
			"transaction_id":      txnId,
			"action":              action,
			"status":              status,
			"created_at":          createdAt,
			"doku_invoice_number": invoiceNum,
			"doku_va_number":      vaNum,
			"payment_amount":      amount,
			"latest_on_action":    latestOnAction,
			"last_updated":        latestAt,
			"seam_stage":          seamStage,
		})
	}
	if orders == nil {
		orders = []map[string]any{}
	}
	return orders, rows.Err()
}

func seamStageFromStatus(status, latestOnAction string) string {
	switch status {
	case "SETTLED":
		return "settled"
	case "RECONCILING":
		return "reconciling"
	case "CONFIRMED":
		return "policy_issued"
	case "INIT_RECEIVED":
		return "va_created"
	}
	if latestOnAction == "on_confirm" {
		return "policy_issued"
	}
	if latestOnAction == "on_init" {
		return "va_created"
	}
	return "va_created"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// stringOrDefault returns the string value of v, or def if v is not a non-empty string.
func stringOrDefault(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// extractOrNew tries to pull an existing transaction_id from the request body
// (at req["transaction_id"] or req["context"]["transaction_id"]) and generates
// new UUIDs for anything not found.
func (s *ClientService) extractOrNew(req map[string]any) (txnId, msgId string) {
	if v, ok := req["transaction_id"].(string); ok && v != "" {
		txnId = v
	} else if ctxMap, ok := req["context"].(map[string]any); ok {
		txnId, _ = ctxMap["transaction_id"].(string)
	}
	if txnId == "" {
		txnId = uuid.NewString()
	}
	msgId = uuid.NewString()
	return
}
