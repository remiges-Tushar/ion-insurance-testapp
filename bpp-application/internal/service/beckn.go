package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ionSchemaBase = "https://raw.githubusercontent.com/remiges-Tushar/ion-specs/refs/heads/feat/FIN-03-motor-insurance-schema/schema/extensions/finance/"

var nikRegex = regexp.MustCompile(`^\d{16}$`)

type BecknService struct {
	db             *pgxpool.Pool
	onixBPPCaller  string
	bppID          string
	ionServiceURL  string
	bapWebhookURL  string
}

func NewBecknService(db *pgxpool.Pool) *BecknService {
	url := os.Getenv("BPP_ONIX_BPP_CALLER_URL")
	if url == "" {
		url = "http://onix-bpp:8082/bpp/caller"
	}
	bppID := os.Getenv("BPP_ID")
	if bppID == "" {
		bppID = "insurance-bpp.iontest"
	}
	ionURL := os.Getenv("ION_SERVICE_URL")
	if ionURL == "" {
		ionURL = "http://ion:8090"
	}
	bapURL := os.Getenv("BAP_WEBHOOK_URL")
	if bapURL == "" {
		bapURL = "http://bap:8083/webhook"
	}
	return &BecknService{db: db, onixBPPCaller: url, bppID: bppID, ionServiceURL: ionURL, bapWebhookURL: bapURL}
}

func (s *BecknService) logMessage(ctx context.Context, action, txnID string, payload map[string]any) {
	payloadJSON, _ := json.Marshal(payload)
	s.db.Exec(ctx,
		`INSERT INTO beckn_message_log (action, transaction_id, payload) VALUES ($1,$2,$3)`,
		action, txnID, payloadJSON,
	)
}

func (s *BecknService) callOnixCaller(action string, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(s.onixBPPCaller+"/"+action, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("call onix-bpp caller %s: %w", action, err)
	}
	defer resp.Body.Close()
	return nil
}

// HandleSearch handles Beckn v2 "discover" action and responds with "on_discover".
func (s *BecknService) HandleSearch(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "discover", txnID, req)

	resources, _ := s.queryMatchingResources(ctx, req)
	response := s.buildOnSearchResponse(ctxData, resources)
	return s.callOnixCaller("on_discover", response)
}

func (s *BecknService) queryMatchingResources(ctx context.Context, req map[string]any) ([]ResourceRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT r.id, r.bpp_id, r.product_type, r.vehicle_type, r.ojk_product_code, r.resource_attributes, r.created_at,
		        COALESCE(o.tariff_zone,''), COALESCE(o.premium_rate_min,0), COALESCE(o.premium_rate_max,0)
		 FROM resources r
		 LEFT JOIN offers o ON o.resource_id = r.id
		 ORDER BY r.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ResourceRow
	for rows.Next() {
		var r ResourceRow
		var attrsJSON []byte
		if err := rows.Scan(&r.ID, &r.BppID, &r.ProductType, &r.VehicleType, &r.OJKProductCode, &attrsJSON, &r.CreatedAt,
			&r.TariffZone, &r.PremiumRateMin, &r.PremiumRateMax); err != nil {
			return nil, err
		}
		json.Unmarshal(attrsJSON, &r.ResourceAttributes)
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *BecknService) buildOnSearchResponse(reqCtx map[string]any, resources []ResourceRow) map[string]any {
	items := make([]any, 0, len(resources))
	for _, r := range resources {
		attrs := make(map[string]any, len(r.ResourceAttributes)+8)
		for k, v := range r.ResourceAttributes {
			attrs[k] = v
		}
		// Always stamp canonical fields from the DB row so the UI can rely on them.
		attrs["@context"] = ionSchemaBase + "insurance-resource/v1/context.jsonld"
		attrs["@type"] = "InsuranceProduct"
		attrs["productType"] = r.ProductType
		attrs["vehicleType"] = r.VehicleType
		attrs["ojkProductCode"] = r.OJKProductCode
		if r.TariffZone != "" {
			attrs["tariffZone"] = r.TariffZone
		}
		if r.PremiumRateMin > 0 {
			attrs["premiumRateMin"] = r.PremiumRateMin
		}
		if r.PremiumRateMax > 0 {
			attrs["premiumRateMax"] = r.PremiumRateMax
		}
		items = append(items, map[string]any{
			"id":                 fmt.Sprintf("%d", r.ID),
			"descriptor":        map[string]any{"name": r.OJKProductCode},
			"resourceAttributes": attrs,
		})
	}
	return map[string]any{
		"context": s.buildResponseContext(reqCtx, "on_discover"),
		"message": map[string]any{
			"catalogs": []any{
				map[string]any{
					"id":         s.bppID,
					"descriptor": map[string]any{"name": s.bppID},
					"provider": map[string]any{
						"id":         s.bppID,
						"descriptor": map[string]any{"name": s.bppID},
					},
					"resources": items,
				},
			},
		},
	}
}

// HandleSelect calculates OJK premium and builds on_select with PolicyQuote.
func (s *BecknService) HandleSelect(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "select", txnID, req)

	msg, _ := req["message"].(map[string]any)
	contract, _ := msg["contract"].(map[string]any)
	commitments, _ := contract["commitments"].([]any)

	var productType, zone string
	var idvIDR int64 = 100_000_000
	if len(commitments) > 0 {
		c, _ := commitments[0].(map[string]any)
		attrs, _ := c["commitmentAttributes"].(map[string]any)
		productType, _ = attrs["productType"].(string)
		zone, _ = attrs["tariffZone"].(string)
		if idvVal, ok := attrs["idv"].(float64); ok {
			idvIDR = int64(idvVal)
		}
	}
	if productType == "" {
		productType = "MOTOR_COMPREHENSIVE"
	}
	if zone == "" {
		zone = "ZONE_3"
	}

	breakup, err := CalcPremium(productType, zone, idvIDR)
	if err != nil {
		breakup = PremiumBreakup{BasePremiumIDR: idvIDR / 30, AdminFeeIDR: adminFeeIDR, StampDutyIDR: stampDutyIDR, TotalIDR: idvIDR/30 + adminFeeIDR + stampDutyIDR, RateUsed: 3.0}
	}

	quoteValidUntil := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	response := map[string]any{
		"context": s.buildResponseContext(ctxData, "on_select"),
		"message": map[string]any{
			"contract": map[string]any{
				"commitments": []any{
					map[string]any{
						"status": map[string]any{"descriptor": map[string]any{"code": "ACTIVE"}},
						"resources": []any{
							map[string]any{
								"id":         "1",
								"descriptor": map[string]any{"name": productType},
								"quantity":   map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"},
							},
						},
						"offer": map[string]any{
							"id":          "quote-" + txnID,
							"resourceIds": []any{"1"},
							"offerAttributes": map[string]any{
								"@context":        ionSchemaBase + "insurance-offer/v1/context.jsonld",
								"@type":           "PolicyQuote",
								"approvedIDV":      idvIDR,
								"annualPremiumIDR": breakup.TotalIDR,
								"rateUsedPct":      breakup.RateUsed,
								"breakup": []any{
									map[string]any{"title": "Base Premium", "amountIDR": breakup.BasePremiumIDR},
									map[string]any{"title": "Admin Fee", "amountIDR": breakup.AdminFeeIDR},
									map[string]any{"title": "Stamp Duty", "amountIDR": breakup.StampDutyIDR},
								},
								"quoteValidUntil": quoteValidUntil,
							},
						},
						"considerationAttributes": map[string]any{
							"@context":       ionSchemaBase + "insurance-consideration/v1/context.jsonld",
							"@type":          "PremiumConsideration",
							"totalAmountIDR": breakup.TotalIDR,
							"breakup": []any{
								map[string]any{"type": "BASE_PREMIUM", "amountIDR": breakup.BasePremiumIDR},
								map[string]any{"type": "ADMIN_FEE", "amountIDR": breakup.AdminFeeIDR},
								map[string]any{"type": "STAMP_DUTY", "amountIDR": breakup.StampDutyIDR},
							},
						},
					},
				},
			},
		},
	}
	return s.callOnixCaller("on_select", response)
}

// HandleInit validates KYC and creates a PENDING_ISSUANCE policy.
func (s *BecknService) HandleInit(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	bapID, _ := ctxData["bap_id"].(string)
	s.logMessage(ctx, "init", txnID, req)

	msg, _ := req["message"].(map[string]any)
	contract, _ := msg["contract"].(map[string]any)

	// Extract KYC from participants[0].participantAttributes
	var nik string
	if parts, ok := contract["participants"].([]any); ok && len(parts) > 0 {
		p, _ := parts[0].(map[string]any)
		attrs, _ := p["participantAttributes"].(map[string]any)
		nik, _ = attrs["nik"].(string)
	}

	// Extract vehicle + IDV from commitments[0].commitmentAttributes
	var vin string
	var idvIDR int64
	if commits, ok := contract["commitments"].([]any); ok && len(commits) > 0 {
		c, _ := commits[0].(map[string]any)
		attrs, _ := c["commitmentAttributes"].(map[string]any)
		vin, _ = attrs["vin"].(string)
		if idv, ok := attrs["idv"].(float64); ok {
			idvIDR = int64(idv)
		}
	}

	if nik != "" && !nikRegex.MatchString(nik) {
		return fmt.Errorf("invalid NIK format: must be 16 digits")
	}

	// Compute premium breakdown for considerationAttributes in the response.
	initBreakup, initErr := CalcPremium("MOTOR_COMPREHENSIVE", "ZONE_3", idvIDR)
	if initErr != nil {
		initBreakup = PremiumBreakup{
			BasePremiumIDR: idvIDR / 30,
			AdminFeeIDR:    adminFeeIDR,
			StampDutyIDR:   stampDutyIDR,
			TotalIDR:       idvIDR/30 + adminFeeIDR + stampDutyIDR,
		}
	}

	var policyID int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO policies (transaction_id, bap_id, bpp_id, status, policyholder_nik, vehicle_vin, idv, doku_invoice_number)
		 VALUES ($1,$2,$3,'PENDING_ISSUANCE',$4,$5,$6,$7)
		 ON CONFLICT (transaction_id) DO UPDATE SET status='PENDING_ISSUANCE', doku_invoice_number=EXCLUDED.doku_invoice_number
		 RETURNING id`,
		txnID, bapID, s.bppID, nik, vin, idvIDR, "ins-"+txnID,
	).Scan(&policyID)
	if err != nil {
		return fmt.Errorf("create policy: %w", err)
	}

	// SEAM: VA creation moved to BAP (via ION service). BPP returns premium + settlementTerms.
	// BAP injects VA/QRIS details into the on_init response before forwarding to frontend.
	fmt.Printf("[SEAM Stage 1] Policy created — invoice=ins-%s policyID=%d (BAP will create VA via ION)\n", txnID, policyID)

	response := map[string]any{
		"context": s.buildResponseContext(ctxData, "on_init"),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnID,
				"commitments": []any{
					map[string]any{
						"status": map[string]any{"descriptor": map[string]any{"code": "ACTIVE"}},
						"resources": []any{
							map[string]any{
								"id":         "1",
								"descriptor": map[string]any{"name": "InsuranceProduct"},
								"quantity":   map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"},
							},
						},
						"offer": map[string]any{"id": "quote-" + txnID, "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context":     ionSchemaBase + "insurance-contract/v1/context.jsonld",
							"@type":        "InsurancePolicy",
							"policyStatus": "PENDING_ISSUANCE",
						},
						"considerationAttributes": map[string]any{
							"@context":       ionSchemaBase + "insurance-consideration/v1/context.jsonld",
							"@type":          "PremiumConsideration",
							"totalAmountIDR": initBreakup.TotalIDR,
							"paymentStatus":  "PENDING",
							"settlementTerms": map[string]any{
								"sellerPct":   97,
								"buyerAppPct": 3,
							},
							"breakup": []any{
								map[string]any{"type": "BASE_PREMIUM", "amountIDR": initBreakup.BasePremiumIDR},
								map[string]any{"type": "ADMIN_FEE", "amountIDR": initBreakup.AdminFeeIDR},
								map[string]any{"type": "STAMP_DUTY", "amountIDR": initBreakup.StampDutyIDR},
							},
						},
					},
				},
			},
		},
	}
	return s.callOnixCaller("on_init", response)
}

// HandleConfirm validates Xendit payment and issues the policy.
func (s *BecknService) HandleConfirm(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "confirm", txnID, req)

	msg, _ := req["message"].(map[string]any)
	contract, _ := msg["contract"].(map[string]any)

	// Extract payment proof from commitments[0].commitmentAttributes.payment (fallback values)
	var paymentMethod, paymentRef string
	var amountPaid int64
	if commits, ok := contract["commitments"].([]any); ok && len(commits) > 0 {
		c, _ := commits[0].(map[string]any)
		attrs, _ := c["commitmentAttributes"].(map[string]any)
		if payment, ok := attrs["payment"].(map[string]any); ok {
			paymentMethod, _ = payment["method"].(string)
			paymentRef, _ = payment["ref"].(string)
			if amt, ok := payment["amount"].(float64); ok {
				amountPaid = int64(amt)
			}
		}
	}

	// Load policy state + DOKU fields
	var policyID int64
	var currentStatus, dokuInvoiceNumber, dokuRequestID string
	var existingPolicyNumber, existingCertURL string
	var existingCoverageStart, existingCoverageEnd *time.Time
	var idvForBreakup int64
	var paymentReceived bool
	err := s.db.QueryRow(ctx,
		`SELECT id, status::text, COALESCE(doku_invoice_number,''), COALESCE(doku_request_id,''), COALESCE(policy_number,''), COALESCE(certificate_url,''), coverage_start, coverage_end, COALESCE(idv,0), COALESCE(payment_received,false)
		 FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&policyID, &currentStatus, &dokuInvoiceNumber, &dokuRequestID, &existingPolicyNumber, &existingCertURL, &existingCoverageStart, &existingCoverageEnd, &idvForBreakup, &paymentReceived)
	if err != nil {
		return fmt.Errorf("policy not found for txn %s: %w", txnID, err)
	}

	// Calculate the correct premium breakdown for the on_confirm response.
	confirmBreakup, _ := CalcPremium("MOTOR_COMPREHENSIVE", "ZONE_3", idvForBreakup)

	var policyNumber, certURL string
	var coverageStart, coverageEnd time.Time

	if currentStatus == "ACTIVE" {
		// Already issued (e.g. by Xendit webhook) — just resend on_confirm.
		policyNumber = existingPolicyNumber
		certURL = existingCertURL
		if existingCoverageStart != nil {
			coverageStart = *existingCoverageStart
		}
		if existingCoverageEnd != nil {
			coverageEnd = *existingCoverageEnd
		}
		// Load payment method/ref from the payment record.
		s.db.QueryRow(ctx,
			`SELECT COALESCE(method,''), COALESCE(payment_ref,''), COALESCE(amount_idr,0) FROM payments WHERE policy_id=$1 ORDER BY paid_at DESC LIMIT 1`,
			policyID,
		).Scan(&paymentMethod, &paymentRef, &amountPaid)
	} else {
		// SEAM Stage 3: payment must have been received (hold set by DOKU → ION → BPP notification)
		// before confirm triggers policy issuance. Settlement release happens at reconcile (Stage 5).
		if !paymentReceived {
			return fmt.Errorf("payment not yet received: complete the bank transfer or use DOKU sandbox")
		}

		fmt.Printf("[SEAM Stage 3] Confirm received — issuing policy (release deferred to reconcile) invoice=%s\n", dokuInvoiceNumber)

		paymentMethod = "DOKU_VA"
		paymentRef = dokuInvoiceNumber

		var seqNum int
		s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE status='ACTIVE'`).Scan(&seqNum)
		seqNum++

		policyNumber = fmt.Sprintf("POL-INS-%d-%05d", time.Now().Year(), seqNum)
		certURL = fmt.Sprintf("https://certificates.insurance.iontest.local/%s.pdf", policyNumber)
		coverageStart = time.Now().UTC()
		coverageEnd = coverageStart.AddDate(1, 0, 0)

		_, err = s.db.Exec(ctx,
			`UPDATE policies SET status='ACTIVE', policy_number=$1, certificate_url=$2, coverage_start=$3, coverage_end=$4 WHERE transaction_id=$5`,
			policyNumber, certURL, coverageStart, coverageEnd, txnID,
		)
		if err != nil {
			return fmt.Errorf("update policy: %w", err)
		}

		s.db.Exec(ctx,
			`INSERT INTO payments (policy_id, method, payment_ref, amount_idr, paid_at) VALUES ($1,$2,$3,$4,NOW())`,
			policyID, paymentMethod, paymentRef, amountPaid,
		)
	}

	response := map[string]any{
		"context": s.buildResponseContext(ctxData, "on_confirm"),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnID,
				"commitments": []any{
					map[string]any{
						"status": map[string]any{"descriptor": map[string]any{"code": "ACTIVE"}},
						"resources": []any{
							map[string]any{
								"id":         "1",
								"descriptor": map[string]any{"name": "InsuranceProduct"},
								"quantity":   map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"},
							},
						},
						"offer": map[string]any{"id": "quote-" + txnID, "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context":       ionSchemaBase + "insurance-contract/v1/context.jsonld",
							"@type":          "InsurancePolicy",
							"policyStatus":   "ACTIVE",
							"policyState":    "ACTIVE",
							"policyNumber":   policyNumber,
							"certificateUrl": certURL,
							"coverageStart":  coverageStart.Format(time.RFC3339),
							"coverageEnd":    coverageEnd.Format(time.RFC3339),
						},
						"considerationAttributes": map[string]any{
							"@context":       ionSchemaBase + "insurance-consideration/v1/context.jsonld",
							"@type":          "PremiumConsideration",
							"totalAmountIDR": confirmBreakup.TotalIDR,
							"paymentMethod":  paymentMethod,
							"paymentRef":     paymentRef,
							"paymentStatus":  "PAID",
							"breakup": []any{
								map[string]any{"type": "BASE_PREMIUM", "amountIDR": confirmBreakup.BasePremiumIDR},
								map[string]any{"type": "ADMIN_FEE", "amountIDR": confirmBreakup.AdminFeeIDR},
								map[string]any{"type": "STAMP_DUTY", "amountIDR": confirmBreakup.StampDutyIDR},
							},
						},
						"performanceAttributes": map[string]any{
							"@context":      ionSchemaBase + "insurance-performance/v1/context.jsonld",
							"@type":         "PolicyPerformance",
							"policyState":   "ACTIVE",
							"coverageStart": coverageStart.Format(time.RFC3339),
							"coverageEnd":   coverageEnd.Format(time.RFC3339),
						},
					},
				},
			},
		},
	}
	return s.callOnixCaller("on_confirm", response)
}

// HandlePaymentReceived records that payment has been received (called by ION service, not Beckn).
// invoiceNumber format: "ins-{transactionID}".
// paymentRequestID is the Request-Id from the DOKU payment webhook (required for settlement release).
func (s *BecknService) HandlePaymentReceived(ctx context.Context, invoiceNumber, paymentRequestID string, amount float64) error {
	if !strings.HasPrefix(invoiceNumber, "ins-") {
		return fmt.Errorf("unexpected invoice_number format: %s", invoiceNumber)
	}
	txnID := strings.TrimPrefix(invoiceNumber, "ins-")

	var policyID int64
	var currentStatus string
	err := s.db.QueryRow(ctx,
		`SELECT id, status::text FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&policyID, &currentStatus)
	if err != nil {
		return fmt.Errorf("policy not found for txn %s: %w", txnID, err)
	}
	if currentStatus == "ACTIVE" {
		return nil // idempotent
	}

	// SEAM Stage 2: funds held at DOKU — policy stays PENDING_ISSUANCE until confirm.
	if paymentRequestID != "" {
		_, err = s.db.Exec(ctx,
			`UPDATE policies SET payment_received=true, doku_request_id=$2 WHERE transaction_id=$1`,
			txnID, paymentRequestID,
		)
	} else {
		_, err = s.db.Exec(ctx,
			`UPDATE policies SET payment_received=true WHERE transaction_id=$1`, txnID,
		)
	}
	if err != nil {
		return fmt.Errorf("mark payment received: %w", err)
	}

	s.db.Exec(ctx,
		`INSERT INTO payments (policy_id, method, payment_ref, amount_idr, paid_at) VALUES ($1,'DOKU_VA',$2,$3,NOW())`,
		policyID, invoiceNumber, int64(amount),
	)
	fmt.Printf("[SEAM Stage 2] Payment received — invoice=%s amount=%.0f funds=HELD policy=PENDING_ISSUANCE\n",
		invoiceNumber, amount)
	return nil
}

// SimulatePayment — sandbox only: directly marks payment_received=true in DB.
// Used when DOKU sandbox portal simulation is not available.
func (s *BecknService) SimulatePayment(ctx context.Context, txnID string) error {
	invoiceNumber := "ins-" + txnID
	var idv int64
	s.db.QueryRow(ctx, `SELECT COALESCE(idv,0) FROM policies WHERE transaction_id=$1`, txnID).Scan(&idv)

	breakup, err := CalcPremium("MOTOR_COMPREHENSIVE", "ZONE_3", idv)
	var amount float64 = 500_000
	if err == nil {
		amount = float64(breakup.TotalIDR)
	}
	return s.HandlePaymentReceived(ctx, invoiceNumber, "", amount)
}

// HandleReconcile processes the Beckn reconcile action from BAP.
// Calls ION service to release held funds with split disbursements, then sends on_reconcile AGREED.
func (s *BecknService) HandleReconcile(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "reconcile", txnID, req)

	msg, _ := req["message"].(map[string]any)
	settlement, _ := msg["settlement"].(map[string]any)
	invoiceNumber, _ := settlement["invoice_number"].(string)
	var totalAmount int64
	if v, ok := settlement["total_amount"].(float64); ok {
		totalAmount = int64(v)
	}

	if invoiceNumber == "" {
		invoiceNumber = "ins-" + txnID
	}

	// Load doku_request_id (payment's Request-Id) needed for DOKU release.
	var dokuRequestID string
	var idv int64
	s.db.QueryRow(ctx,
		`SELECT COALESCE(doku_request_id,''), COALESCE(idv,0) FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&dokuRequestID, &idv)

	// Use IDV-based premium if total_amount not provided in reconcile message.
	if totalAmount == 0 && idv > 0 {
		if bp, err := CalcPremium("MOTOR_COMPREHENSIVE", "ZONE_3", idv); err == nil {
			totalAmount = bp.TotalIDR
		}
	}

	// Call ION service to execute the DOKU settlement release with splits.
	bppBankAcctID := os.Getenv("DOKU_BPP_BANK_ACCOUNT_ID")
	bapBankAcctID := os.Getenv("DOKU_BAP_BANK_ACCOUNT_ID")

	splits := []map[string]any{}
	if bppBankAcctID != "" {
		splits = append(splits, map[string]any{
			"bank_account_settlement_id": bppBankAcctID,
			"value":                      97.0,
			"type":                       "PERCENTAGE",
		})
	}
	if bapBankAcctID != "" {
		splits = append(splits, map[string]any{
			"bank_account_settlement_id": bapBankAcctID,
			"value":                      3.0,
			"type":                       "PERCENTAGE",
		})
	}

	fmt.Printf("[SEAM Stage 5] Calling ION release — invoice=%s originalReqID=%s amount=%d splits=%d\n",
		invoiceNumber, dokuRequestID, totalAmount, len(splits))

	releasePayload := map[string]any{
		"invoice_number":      invoiceNumber,
		"original_request_id": dokuRequestID,
		"amount":              totalAmount,
		"splits":              splits,
	}
	if rerr := s.callIONRelease(releasePayload); rerr != nil {
		// Non-fatal in sandbox — log and continue so on_reconcile is still sent.
		fmt.Printf("[SEAM Stage 5] ION release failed (non-fatal in sandbox): %v\n", rerr)
	} else {
		fmt.Printf("[SEAM Stage 5] ION release SUCCESS\n")
	}

	// Mark reconcile settled.
	s.db.Exec(ctx, `UPDATE policies SET reconcile_status='SETTLED' WHERE transaction_id=$1`, txnID)

	// Send on_reconcile AGREED.
	response := map[string]any{
		"context": s.buildResponseContext(ctxData, "on_reconcile"),
		"message": map[string]any{
			"settlement": map[string]any{
				"status":         "AGREED",
				"invoice_number": invoiceNumber,
			},
		},
	}
	fmt.Printf("[SEAM Stage 5] Sending on_reconcile AGREED directly to BAP — txn=%s\n", txnID)
	return s.postDirect(s.bapWebhookURL+"/on_reconcile", response)
}

func (s *BecknService) postDirect(url string, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
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

func (s *BecknService) callIONRelease(payload map[string]any) error {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(s.ionServiceURL+"/settlement/release", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("call ION release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ION release HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetPaymentStatus returns SEAM-relevant payment state for a transaction (used by frontend polling).
func (s *BecknService) GetPaymentStatus(ctx context.Context, txnID string) (map[string]any, error) {
	var policyStatus, reconcileStatus string
	var paymentReceived bool
	var dokuInvoiceNumber, dokuVANumber string
	var amountIDR int64
	err := s.db.QueryRow(ctx,
		`SELECT status::text, COALESCE(payment_received,false), COALESCE(doku_invoice_number,''), COALESCE(doku_va_number,''), COALESCE(idv,0), COALESCE(reconcile_status,'PENDING')
		 FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&policyStatus, &paymentReceived, &dokuInvoiceNumber, &dokuVANumber, &amountIDR, &reconcileStatus)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	seamStage := "Stage 1 — VA Created (awaiting payment)"
	if paymentReceived && policyStatus != "ACTIVE" {
		seamStage = "Stage 2 — Funds Held at DOKU (awaiting confirm)"
	} else if policyStatus == "ACTIVE" && reconcileStatus != "SETTLED" {
		seamStage = "Stage 3+4 — Policy Active (reconcile pending)"
	} else if reconcileStatus == "SETTLED" {
		seamStage = "Stage 5 — Settled (funds released with splits)"
	}

	return map[string]any{
		"transaction_id":      txnID,
		"policy_status":       policyStatus,
		"payment_received":    paymentReceived,
		"doku_invoice_number": dokuInvoiceNumber,
		"doku_va_number":      dokuVANumber,
		"reconcile_status":    reconcileStatus,
		"seam_stage":          seamStage,
	}, nil
}

// HandleStatus returns current policy state.
func (s *BecknService) HandleStatus(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "status", txnID, req)

	var status, policyNumber, certURL string
	var coverageStart, coverageEnd *time.Time
	err := s.db.QueryRow(ctx,
		`SELECT status, COALESCE(policy_number,''), COALESCE(certificate_url,''), coverage_start, coverage_end
		 FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&status, &policyNumber, &certURL, &coverageStart, &coverageEnd)

	orderStatus := "UNKNOWN"
	if err == nil {
		orderStatus = status
	}

	response := map[string]any{
		"context": s.buildResponseContext(ctxData, "on_status"),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnID,
				"commitments": []any{
					map[string]any{
						"status": map[string]any{"descriptor": map[string]any{"code": orderStatus}},
						"resources": []any{
							map[string]any{
								"id":         "1",
								"descriptor": map[string]any{"name": "InsuranceProduct"},
								"quantity":   map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"},
							},
						},
						"offer": map[string]any{"id": "quote-" + txnID, "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context":       ionSchemaBase + "insurance-contract/v1/context.jsonld",
							"@type":          "InsurancePolicy",
							"policyStatus":   orderStatus,
							"policyNumber":   policyNumber,
							"certificateUrl": certURL,
						},
						"performanceAttributes": map[string]any{
							"@context":       ionSchemaBase + "insurance-performance/v1/context.jsonld",
							"@type":          "PolicyPerformance",
							"policyState":    orderStatus,
							"policyNumber":   policyNumber,
							"certificateUrl": certURL,
						},
					},
				},
			},
		},
	}
	return s.callOnixCaller("on_status", response)
}

// HandleCancel cancels a pre-issuance policy.
func (s *BecknService) HandleCancel(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "cancel", txnID, req)

	s.db.Exec(ctx,
		`UPDATE policies SET status='CANCELLED' WHERE transaction_id=$1 AND status='PENDING_ISSUANCE'`,
		txnID,
	)

	response := map[string]any{
		"context": s.buildResponseContext(ctxData, "on_cancel"),
		"message": map[string]any{
			"contract": map[string]any{
				"id": txnID,
				"commitments": []any{
					map[string]any{
						"status": map[string]any{"descriptor": map[string]any{"code": "CLOSED"}},
						"resources": []any{
							map[string]any{
								"id":         "1",
								"descriptor": map[string]any{"name": "InsuranceProduct"},
								"quantity":   map[string]any{"unitQuantity": 1, "unitCode": "policy", "unitText": "policy"},
							},
						},
						"offer": map[string]any{"id": "quote-" + txnID, "resourceIds": []any{"1"}},
						"commitmentAttributes": map[string]any{
							"@context":     ionSchemaBase + "insurance-contract/v1/context.jsonld",
							"@type":        "InsurancePolicy",
							"policyStatus": "CANCELLED",
						},
					},
				},
			},
		},
	}
	return s.callOnixCaller("on_cancel", response)
}

// HandleRate saves a rating.
func (s *BecknService) HandleRate(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "rate", txnID, req)

	msg, _ := req["message"].(map[string]any)
	var score int
	var feedback string
	if inputs, ok := msg["ratingInputs"].([]any); ok && len(inputs) > 0 {
		input, _ := inputs[0].(map[string]any)
		if rangeObj, ok := input["range"].(map[string]any); ok {
			if v, ok := rangeObj["value"].(float64); ok {
				score = int(v)
			}
		}
		if ffs, ok := input["feedbackFormSubmission"].(map[string]any); ok {
			if data, ok := ffs["data"].(map[string]any); ok {
				feedback, _ = data["feedback"].(string)
			}
		}
	}

	var policyID int64
	s.db.QueryRow(ctx, `SELECT id FROM policies WHERE transaction_id=$1`, txnID).Scan(&policyID)
	if policyID > 0 {
		s.db.Exec(ctx, `INSERT INTO ratings (policy_id, score, feedback) VALUES ($1,$2,$3)`, policyID, score, feedback)
	}

	return s.callOnixCaller("on_rate", map[string]any{
		"context": s.buildResponseContext(ctxData, "on_rate"),
		"message": map[string]any{
			"ratings": []any{
				map[string]any{
					"target": map[string]any{
						"id": txnID,
					},
					"range": map[string]any{},
				},
			},
		},
	})
}

// HandleSupport creates a support ticket.
func (s *BecknService) HandleSupport(ctx context.Context, req map[string]any) error {
	ctxData, _ := req["context"].(map[string]any)
	txnID, _ := ctxData["transaction_id"].(string)
	s.logMessage(ctx, "support", txnID, req)

	msg, _ := req["message"].(map[string]any)
	var desc string
	if support, ok := msg["support"].(map[string]any); ok {
		if d, ok := support["descriptor"].(map[string]any); ok {
			desc, _ = d["name"].(string)
		}
	}

	var policyID *int64
	var pid int64
	if s.db.QueryRow(ctx, `SELECT id FROM policies WHERE transaction_id=$1`, txnID).Scan(&pid) == nil && pid > 0 {
		policyID = &pid
	}
	if desc == "" {
		desc = "Support request"
	}
	s.db.Exec(ctx, `INSERT INTO support_tickets (policy_id, description) VALUES ($1,$2)`, policyID, desc)

	return s.callOnixCaller("on_support", map[string]any{
		"context": s.buildResponseContext(ctxData, "on_support"),
		"message": map[string]any{
			"support": map[string]any{
				"orderId": txnID,
				"descriptor": map[string]any{
					"name": "Support ticket created. Our team will contact you shortly.",
				},
				"channels": []any{
					map[string]any{"type": "EMAIL"},
				},
			},
		},
	})
}

func (s *BecknService) buildResponseContext(reqCtx map[string]any, action string) map[string]any {
	bapID, _ := reqCtx["bap_id"].(string)
	bapURI, _ := reqCtx["bap_uri"].(string)
	txnID, _ := reqCtx["transaction_id"].(string)
	version, _ := reqCtx["version"].(string)
	if version == "" {
		version = "2.0.0"
	}
	return map[string]any{
		"version":        version,
		"action":         action,
		"domain":         "ion:finance",
		"bap_id":         bapID,
		"bap_uri":        bapURI,
		"bpp_id":         s.bppID,
		"bpp_uri":        "http://onix-bpp:8082/bpp/receiver",
		"transaction_id": txnID,
		"message_id":     fmt.Sprintf("resp-%s-%d", txnID, time.Now().UnixNano()),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"ttl":            "PT30S",
	}
}

// Dashboard / list methods

type PolicyRow struct {
	ID                int64      `json:"id"`
	TransactionID     string     `json:"transaction_id"`
	BapID             string     `json:"bap_id"`
	Status            string     `json:"status"`
	PolicyholderNIK   string     `json:"policyholder_nik"`
	VehicleVIN        string     `json:"vehicle_vin"`
	IDV               int64      `json:"idv"`
	PolicyNumber      string     `json:"policy_number"`
	CertificateURL    string     `json:"certificate_url"`
	CoverageStart     *time.Time `json:"coverage_start"`
	CoverageEnd       *time.Time `json:"coverage_end"`
	CreatedAt         time.Time  `json:"created_at"`
	PaymentReceived   bool       `json:"payment_received"`
	DokuInvoiceNumber string     `json:"doku_invoice_number"`
	DokuVANumber      string     `json:"doku_va_number"`
	ReconcileStatus   string     `json:"reconcile_status"`
}

func (s *BecknService) ListPolicies(ctx context.Context) ([]PolicyRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, transaction_id, bap_id, status::text, COALESCE(policyholder_nik,''), COALESCE(vehicle_vin,''),
		        COALESCE(idv,0), COALESCE(policy_number,''), COALESCE(certificate_url,''),
		        coverage_start, coverage_end, created_at,
		        COALESCE(payment_received,false), COALESCE(doku_invoice_number,''),
		        COALESCE(doku_va_number,''), COALESCE(reconcile_status,'PENDING')
		 FROM policies ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []PolicyRow
	for rows.Next() {
		var p PolicyRow
		if err := rows.Scan(&p.ID, &p.TransactionID, &p.BapID, &p.Status, &p.PolicyholderNIK,
			&p.VehicleVIN, &p.IDV, &p.PolicyNumber, &p.CertificateURL,
			&p.CoverageStart, &p.CoverageEnd, &p.CreatedAt,
			&p.PaymentReceived, &p.DokuInvoiceNumber, &p.DokuVANumber, &p.ReconcileStatus); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func (s *BecknService) GetPolicy(ctx context.Context, id int64) (*PolicyRow, error) {
	p := &PolicyRow{}
	err := s.db.QueryRow(ctx,
		`SELECT id, transaction_id, bap_id, status::text, COALESCE(policyholder_nik,''), COALESCE(vehicle_vin,''),
		        COALESCE(idv,0), COALESCE(policy_number,''), COALESCE(certificate_url,''),
		        coverage_start, coverage_end, created_at,
		        COALESCE(payment_received,false), COALESCE(doku_invoice_number,''),
		        COALESCE(doku_va_number,''), COALESCE(reconcile_status,'PENDING')
		 FROM policies WHERE id=$1`, id,
	).Scan(&p.ID, &p.TransactionID, &p.BapID, &p.Status, &p.PolicyholderNIK,
		&p.VehicleVIN, &p.IDV, &p.PolicyNumber, &p.CertificateURL,
		&p.CoverageStart, &p.CoverageEnd, &p.CreatedAt,
		&p.PaymentReceived, &p.DokuInvoiceNumber, &p.DokuVANumber, &p.ReconcileStatus)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *BecknService) GetDashboardStats(ctx context.Context) (map[string]any, error) {
	var activePolicies, policiesIssued, premiumsCollected, claimsFiled, claimsPending, supportTickets, renewals int64
	var avgPremium float64
	var paymentPending, paymentReceived, reconcileSettled, reconcilePending int64

	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE status='ACTIVE'`).Scan(&activePolicies)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies`).Scan(&policiesIssued)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount_idr),0) FROM payments`).Scan(&premiumsCollected)
	s.db.QueryRow(ctx, `SELECT COALESCE(AVG(amount_idr),0.0) FROM payments`).Scan(&avgPremium)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM claims`).Scan(&claimsFiled)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM claims WHERE status='FILED'`).Scan(&claimsPending)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM support_tickets WHERE status='OPEN'`).Scan(&supportTickets)
	s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM policies WHERE status='ACTIVE' AND coverage_end BETWEEN NOW() AND NOW() + INTERVAL '30 days'`,
	).Scan(&renewals)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE COALESCE(payment_received,false)=false AND status='PENDING_ISSUANCE'`).Scan(&paymentPending)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE COALESCE(payment_received,false)=true`).Scan(&paymentReceived)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE COALESCE(reconcile_status,'PENDING')='SETTLED'`).Scan(&reconcileSettled)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE COALESCE(reconcile_status,'PENDING')!='SETTLED' AND status='ACTIVE'`).Scan(&reconcilePending)

	return map[string]any{
		"active_policies":        activePolicies,
		"policies_issued":        policiesIssued,
		"premiums_collected_idr": premiumsCollected,
		"avg_premium_idr":        int64(avgPremium),
		"claims_filed":           claimsFiled,
		"claims_pending":         claimsPending,
		"support_tickets":        supportTickets,
		"renewals_next_30_days":  renewals,
		"seam_payment_pending":   paymentPending,
		"seam_payment_received":  paymentReceived,
		"seam_reconcile_settled": reconcileSettled,
		"seam_reconcile_pending": reconcilePending,
	}, nil
}

type ClaimRow struct {
	ID                  int64     `json:"id"`
	PolicyID            int64     `json:"policy_id"`
	ClaimID             string    `json:"claim_id"`
	IncidentType        string    `json:"incident_type"`
	Status              string    `json:"status"`
	EstimatedDamageIDR  int64     `json:"estimated_damage_idr"`
	FiledAt             time.Time `json:"filed_at"`
}

func (s *BecknService) ListClaims(ctx context.Context) ([]ClaimRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, policy_id, claim_id, COALESCE(incident_type,''), status, COALESCE(estimated_damage_idr,0), filed_at
		 FROM claims ORDER BY filed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ClaimRow
	for rows.Next() {
		var c ClaimRow
		if err := rows.Scan(&c.ID, &c.PolicyID, &c.ClaimID, &c.IncidentType, &c.Status, &c.EstimatedDamageIDR, &c.FiledAt); err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, rows.Err()
}

type MessageLogRow struct {
	ID            int64          `json:"id"`
	Action        string         `json:"action"`
	TransactionID string         `json:"transaction_id"`
	Payload       map[string]any `json:"payload"`
	CreatedAt     time.Time      `json:"created_at"`
}

func (s *BecknService) ListMessages(ctx context.Context) ([]MessageLogRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, action, COALESCE(transaction_id,''), payload, created_at FROM beckn_message_log ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []MessageLogRow
	for rows.Next() {
		var m MessageLogRow
		var payloadJSON []byte
		if err := rows.Scan(&m.ID, &m.Action, &m.TransactionID, &payloadJSON, &m.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(payloadJSON, &m.Payload)
		results = append(results, m)
	}
	return results, rows.Err()
}

type SupportTicketRow struct {
	ID            int64     `json:"id"`
	PolicyID      *int64    `json:"policy_id"`
	PolicyNumber  string    `json:"policy_number"`
	TransactionID string    `json:"transaction_id"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *BecknService) ListSupportTickets(ctx context.Context) ([]SupportTicketRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT st.id, st.policy_id,
		        COALESCE(p.policy_number,''),
		        COALESCE(p.transaction_id,''),
		        st.description, st.status, st.created_at
		 FROM support_tickets st
		 LEFT JOIN policies p ON p.id = st.policy_id
		 ORDER BY st.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []SupportTicketRow
	for rows.Next() {
		var t SupportTicketRow
		if err := rows.Scan(&t.ID, &t.PolicyID, &t.PolicyNumber, &t.TransactionID, &t.Description, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	return results, rows.Err()
}

type RatingRow struct {
	ID            int64     `json:"id"`
	PolicyID      *int64    `json:"policy_id"`
	PolicyNumber  string    `json:"policy_number"`
	TransactionID string    `json:"transaction_id"`
	Score         int       `json:"score"`
	Feedback      string    `json:"feedback"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *BecknService) ListRatings(ctx context.Context) ([]RatingRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT r.id, r.policy_id,
		        COALESCE(p.policy_number,''),
		        COALESCE(p.transaction_id,''),
		        r.score, COALESCE(r.feedback,''), r.created_at
		 FROM ratings r
		 LEFT JOIN policies p ON p.id = r.policy_id
		 ORDER BY r.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []RatingRow
	for rows.Next() {
		var r RatingRow
		if err := rows.Scan(&r.ID, &r.PolicyID, &r.PolicyNumber, &r.TransactionID, &r.Score, &r.Feedback, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
