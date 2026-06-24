package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	db            *pgxpool.Pool
	onixBPPCaller string
	bppID         string
	xendit        *XenditService
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
	return &BecknService{db: db, onixBPPCaller: url, bppID: bppID, xendit: NewXenditService()}
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
		`INSERT INTO policies (transaction_id, bap_id, bpp_id, status, policyholder_nik, vehicle_vin, idv)
		 VALUES ($1,$2,$3,'PENDING_ISSUANCE',$4,$5,$6)
		 ON CONFLICT (transaction_id) DO UPDATE SET status='PENDING_ISSUANCE'
		 RETURNING id`,
		txnID, bapID, s.bppID, nik, vin, idvIDR,
	).Scan(&policyID)
	if err != nil {
		return fmt.Errorf("create policy: %w", err)
	}

	// Create Xendit VA + QRIS; fall back to mock values if Xendit is unavailable.
	vaAccountNumber := fmt.Sprintf("8800%010d", policyID)
	bankCode := "MOCK"
	qrisString := ""
	if s.xendit != nil && s.xendit.secretKey != "" {
		breakup, calcErr := CalcPremium("MOTOR_COMPREHENSIVE", "ZONE_3", idvIDR)
		totalPremium := int64(500_000)
		if calcErr == nil {
			totalPremium = breakup.TotalIDR
		} else if idvIDR > 0 {
			totalPremium = idvIDR/30 + adminFeeIDR + stampDutyIDR
		}

		if va, xerr := s.xendit.CreateVirtualAccount("ins-"+txnID, "ION Insurance", totalPremium); xerr == nil && va.AccountNumber != "" {
			s.db.Exec(ctx, `UPDATE policies SET xendit_va_id=$1, bank_code=$2 WHERE id=$3`, va.ID, va.BankCode, policyID)
			vaAccountNumber = va.AccountNumber
			bankCode = va.BankCode
		}

		if qr, qerr := s.xendit.CreateQRIS("ins-"+txnID+"-qris", totalPremium); qerr == nil {
			s.db.Exec(ctx, `UPDATE policies SET xendit_qris_id=$1, xendit_qris_string=$2 WHERE id=$3`, qr.ID, qr.QRString, policyID)
			qrisString = qr.QRString
		}
	}

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
							"virtualAccount": vaAccountNumber,
							"bankCode":       bankCode,
							"qrisString":     qrisString,
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

	// Load policy state + Xendit IDs
	var policyID int64
	var currentStatus, xenditVAID, xenditQRISID string
	var existingPolicyNumber, existingCertURL string
	var existingCoverageStart, existingCoverageEnd *time.Time
	var idvForBreakup int64
	err := s.db.QueryRow(ctx,
		`SELECT id, status::text, COALESCE(xendit_va_id,''), COALESCE(xendit_qris_id,''), COALESCE(policy_number,''), COALESCE(certificate_url,''), coverage_start, coverage_end, COALESCE(idv,0)
		 FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&policyID, &currentStatus, &xenditVAID, &xenditQRISID, &existingPolicyNumber, &existingCertURL, &existingCoverageStart, &existingCoverageEnd, &idvForBreakup)
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
		// Verify payment via Xendit based on the method the user chose.
		if s.xendit != nil {
			if paymentMethod == "QRIS" && xenditQRISID != "" {
				payments, verr := s.xendit.GetQRISPayments(xenditQRISID)
				if verr != nil || len(payments) == 0 {
					return fmt.Errorf("QRIS payment not yet confirmed by Xendit: scan and pay the QR code first")
				}
				amountPaid = int64(payments[0].Amount)
				paymentRef = xenditQRISID
			} else if xenditVAID != "" {
				payments, verr := s.xendit.GetVAPayments(xenditVAID)
				if verr != nil || len(payments) == 0 {
					return fmt.Errorf("VA payment not yet confirmed by Xendit: simulate payment in Xendit sandbox first")
				}
				amountPaid = int64(payments[0].Amount)
				paymentMethod = "XENDIT_VA"
				paymentRef = xenditVAID
			}
		}

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

// VerifyXenditToken checks the Xendit webhook callback token.
func (s *BecknService) VerifyXenditToken(token string) bool {
	if s.xendit == nil {
		return false
	}
	return s.xendit.VerifyToken(token)
}

// ProcessXenditPayment marks the policy as ACTIVE when Xendit's payment webhook fires.
// externalID format: "ins-{transactionID}".
func (s *BecknService) ProcessXenditPayment(ctx context.Context, externalID string, amount float64, bankCode string) error {
	if !strings.HasPrefix(externalID, "ins-") {
		return fmt.Errorf("unexpected xendit external_id format: %s", externalID)
	}
	txnID := strings.TrimPrefix(externalID, "ins-")

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

	var seqNum int
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE status='ACTIVE'`).Scan(&seqNum)
	seqNum++

	policyNumber := fmt.Sprintf("POL-INS-%d-%05d", time.Now().Year(), seqNum)
	certURL := fmt.Sprintf("https://certificates.insurance.iontest.local/%s.pdf", policyNumber)
	coverageStart := time.Now().UTC()
	coverageEnd := coverageStart.AddDate(1, 0, 0)

	_, err = s.db.Exec(ctx,
		`UPDATE policies SET status='ACTIVE', policy_number=$1, certificate_url=$2, coverage_start=$3, coverage_end=$4 WHERE transaction_id=$5`,
		policyNumber, certURL, coverageStart, coverageEnd, txnID,
	)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}

	s.db.Exec(ctx,
		`INSERT INTO payments (policy_id, method, payment_ref, amount_idr, paid_at) VALUES ($1,$2,$3,$4,NOW())`,
		policyID, "XENDIT_VA", externalID, int64(amount),
	)
	return nil
}

// ProcessXenditQRISPayment marks the policy ACTIVE when a QRIS payment webhook fires.
// referenceID format: "ins-{transactionID}-qris".
func (s *BecknService) ProcessXenditQRISPayment(ctx context.Context, referenceID string, amount float64) error {
	if !strings.HasPrefix(referenceID, "ins-") || !strings.HasSuffix(referenceID, "-qris") {
		return fmt.Errorf("unexpected QRIS reference_id format: %s", referenceID)
	}
	txnID := strings.TrimSuffix(strings.TrimPrefix(referenceID, "ins-"), "-qris")

	var policyID int64
	var currentStatus string
	err := s.db.QueryRow(ctx,
		`SELECT id, status::text FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&policyID, &currentStatus)
	if err != nil {
		return fmt.Errorf("policy not found for txn %s: %w", txnID, err)
	}
	if currentStatus == "ACTIVE" {
		return nil
	}

	var seqNum int
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM policies WHERE status='ACTIVE'`).Scan(&seqNum)
	seqNum++

	policyNumber := fmt.Sprintf("POL-INS-%d-%05d", time.Now().Year(), seqNum)
	certURL := fmt.Sprintf("https://certificates.insurance.iontest.local/%s.pdf", policyNumber)
	coverageStart := time.Now().UTC()
	coverageEnd := coverageStart.AddDate(1, 0, 0)

	_, err = s.db.Exec(ctx,
		`UPDATE policies SET status='ACTIVE', policy_number=$1, certificate_url=$2, coverage_start=$3, coverage_end=$4 WHERE transaction_id=$5`,
		policyNumber, certURL, coverageStart, coverageEnd, txnID,
	)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}

	s.db.Exec(ctx,
		`INSERT INTO payments (policy_id, method, payment_ref, amount_idr, paid_at) VALUES ($1,$2,$3,$4,NOW())`,
		policyID, "XENDIT_QRIS", referenceID, int64(amount),
	)
	return nil
}

// SimulatePayment triggers a Xendit sandbox payment for testing.
// method: "VA" or "QRIS"
func (s *BecknService) SimulatePayment(ctx context.Context, txnID, method string) error {
	if s.xendit == nil || s.xendit.secretKey == "" {
		return fmt.Errorf("xendit not configured")
	}

	var xenditVAID, xenditQRISID string
	var idv int64
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(xendit_va_id,''), COALESCE(xendit_qris_id,''), COALESCE(idv,0)
		 FROM policies WHERE transaction_id=$1`, txnID,
	).Scan(&xenditVAID, &xenditQRISID, &idv)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	breakup, calcErr := CalcPremium("MOTOR_COMPREHENSIVE", "ZONE_3", idv)
	var amount int64 = 500_000
	if calcErr == nil {
		amount = breakup.TotalIDR
	}

	if method == "QRIS" {
		if xenditQRISID == "" {
			return fmt.Errorf("no QRIS ID on record for this policy")
		}
		return s.xendit.SimulateQRISPayment(xenditQRISID, amount)
	}
	return s.xendit.SimulateVAPayment("ins-"+txnID, amount)
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
	ID              int64      `json:"id"`
	TransactionID   string     `json:"transaction_id"`
	BapID           string     `json:"bap_id"`
	Status          string     `json:"status"`
	PolicyholderNIK string     `json:"policyholder_nik"`
	VehicleVIN      string     `json:"vehicle_vin"`
	IDV             int64      `json:"idv"`
	PolicyNumber    string     `json:"policy_number"`
	CertificateURL  string     `json:"certificate_url"`
	CoverageStart   *time.Time `json:"coverage_start"`
	CoverageEnd     *time.Time `json:"coverage_end"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (s *BecknService) ListPolicies(ctx context.Context) ([]PolicyRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, transaction_id, bap_id, status::text, COALESCE(policyholder_nik,''), COALESCE(vehicle_vin,''),
		        COALESCE(idv,0), COALESCE(policy_number,''), COALESCE(certificate_url,''),
		        coverage_start, coverage_end, created_at
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
			&p.CoverageStart, &p.CoverageEnd, &p.CreatedAt); err != nil {
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
		        coverage_start, coverage_end, created_at
		 FROM policies WHERE id=$1`, id,
	).Scan(&p.ID, &p.TransactionID, &p.BapID, &p.Status, &p.PolicyholderNIK,
		&p.VehicleVIN, &p.IDV, &p.PolicyNumber, &p.CertificateURL,
		&p.CoverageStart, &p.CoverageEnd, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *BecknService) GetDashboardStats(ctx context.Context) (map[string]any, error) {
	var activePolicies, policiesIssued, premiumsCollected, claimsFiled, claimsPending, supportTickets, renewals int64
	var avgPremium float64

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

	return map[string]any{
		"active_policies":      activePolicies,
		"policies_issued":      policiesIssued,
		"premiums_collected_idr": premiumsCollected,
		"avg_premium_idr":      int64(avgPremium),
		"claims_filed":         claimsFiled,
		"claims_pending":       claimsPending,
		"support_tickets":      supportTickets,
		"renewals_next_30_days": renewals,
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
