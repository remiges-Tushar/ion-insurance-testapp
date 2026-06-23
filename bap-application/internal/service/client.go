package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClientService is the core business-logic layer for the BAP application.
// It mediates between the frontend-facing HTTP handlers and the Beckn network
// via onix-bap, using Redis pub/sub to turn async Beckn callbacks into
// synchronous frontend responses.
type ClientService struct {
	db               *pgxpool.Pool
	cb               *CallbackManager
	onixBAPCallerURL string
}

// NewClientService constructs a ClientService.
// onixBAPCallerURL is typically read from env BAP_ONIX_BAP_CALLER_URL.
func NewClientService(db *pgxpool.Pool, cb *CallbackManager, onixBAPCallerURL string) *ClientService {
	return &ClientService{
		db:               db,
		cb:               cb,
		onixBAPCallerURL: onixBAPCallerURL,
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
						"resources": []any{map[string]any{"@context": "https://schema.beckn.one/ion/finance/v1", "@type": "beckn:Resource", "id": productId, "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{productId}},
						"commitmentAttributes": map[string]any{
							"@context":    "https://schema.beckn.one/ion/finance/v1",
							"@type":       "ion:InsuranceCommitment",
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
							"@context": "https://schema.beckn.one/ion/finance/v1",
							"@type":    "ion:Policyholder",
							"nik":      nik,
						},
					},
				},
				"commitments": []any{
					map[string]any{
						"status":    map[string]any{"descriptor": map[string]any{"code": "DRAFT"}},
						"resources": []any{map[string]any{"@context": "https://schema.beckn.one/ion/finance/v1", "@type": "beckn:Resource", "id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context": "https://schema.beckn.one/ion/finance/v1",
							"@type":    "ion:InsuranceCommitment",
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
						"resources": []any{map[string]any{"@context": "https://schema.beckn.one/ion/finance/v1", "@type": "beckn:Resource", "id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
						"offer":     map[string]any{"id": "1", "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context": "https://schema.beckn.one/ion/finance/v1",
							"@type":    "ion:InsuranceCommitment",
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
	return parseResponse(body)
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
						"resources": []any{map[string]any{"@context": "https://schema.beckn.one/ion/finance/v1", "@type": "beckn:Resource", "id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
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
						"resources": []any{map[string]any{"@context": "https://schema.beckn.one/ion/finance/v1", "@type": "beckn:Resource", "id": "1", "descriptor": map[string]any{"name": "InsuranceProduct"}, "quantity": map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"}}},
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
					map[string]any{
						"@context": "https://schema.beckn.io/",
						"@type":    "beckn:SupportChannel",
					},
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

	// Persist snapshot and audit log.
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
		s.upsertTransaction(ctx, txnId, "confirm", "CONFIRMED")
	case "on_cancel":
		s.upsertTransaction(ctx, txnId, "cancel", "CANCELLED")
	// on_search, on_status, on_rate, on_support — no status change
	}

	// Publish to unblock the waiting forwardAndWait call.
	if err := s.cb.Publish(ctx, route, txnId, msgId, data); err != nil {
		// Not fatal — the frontend will time out gracefully.
		log.Printf("[ClientService] Publish failed for onAction=%s txnId=%s: %v", onAction, txnId, err)
	}
	return nil
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
