package api

import "time"

// Auth

type RegisterRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	OJKLicense  string `json:"ojk_license" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// Catalog / Inventory

type CreateProviderRequest struct {
	Name               string         `json:"name" binding:"required"`
	Descriptor         map[string]any `json:"descriptor"`
	Locations          []any          `json:"locations"`
	ProviderAttributes map[string]any `json:"provider_attributes"`
}

type CreateCatalogRequest struct {
	Name       string         `json:"name" binding:"required"`
	Descriptor map[string]any `json:"descriptor"`
	Validity   map[string]any `json:"validity"`
	Version    string         `json:"version"`
	ProviderID *int64         `json:"provider_id"`
}

type CreateResourceRequest struct {
	ProductType        string         `json:"product_type" binding:"required"`
	VehicleType        string         `json:"vehicle_type" binding:"required"`
	OJKProductCode     string         `json:"ojk_product_code" binding:"required"`
	ResourceAttributes map[string]any `json:"resource_attributes"`
}

type CreateOfferRequest struct {
	ResourceID      int64          `json:"resource_id" binding:"required"`
	TariffZone      string         `json:"tariff_zone" binding:"required"`
	PremiumRateMin  float64        `json:"premium_rate_min" binding:"required"`
	PremiumRateMax  float64        `json:"premium_rate_max" binding:"required"`
	OfferAttributes map[string]any `json:"offer_attributes"`
	ValidUntil      *time.Time     `json:"valid_until"`
}

type PublishCatalogRequest struct {
	CatalogID int64 `json:"catalog_id" binding:"required"`
}

// Dashboard

type DashboardStats struct {
	PoliciesIssued     int64 `json:"policies_issued"`
	ActivePolicies     int64 `json:"active_policies"`
	PremiumsCollected  int64 `json:"premiums_collected_idr"`
	AvgPremium         int64 `json:"avg_premium_idr"`
	ClaimsFiled        int64 `json:"claims_filed"`
	ClaimsPending      int64 `json:"claims_pending"`
	RenewalsNext30Days int64 `json:"renewals_next_30_days"`
	SupportTickets     int64 `json:"support_tickets"`
}

// Beckn message types

type BecknContext struct {
	Version       string         `json:"version"`
	Action        string         `json:"action"`
	Domain        string         `json:"domain"`
	BapID         string         `json:"bap_id"`
	BapURI        string         `json:"bap_uri"`
	BppID         string         `json:"bpp_id"`
	BppURI        string         `json:"bpp_uri"`
	TransactionID string         `json:"transaction_id"`
	MessageID     string         `json:"message_id"`
	Timestamp     string         `json:"timestamp"`
	TTL           string         `json:"ttl"`
	Location      map[string]any `json:"location,omitempty"`
}

type BecknRequest struct {
	Context BecknContext    `json:"context"`
	Message map[string]any  `json:"message"`
}

type AckResponse struct {
	Message struct {
		Ack struct {
			Status string `json:"status"`
		} `json:"ack"`
	} `json:"message"`
}

// Policy

type PolicyResponse struct {
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

// Message log

type MessageLogEntry struct {
	ID            int64          `json:"id"`
	Action        string         `json:"action"`
	TransactionID string         `json:"transaction_id"`
	Payload       map[string]any `json:"payload"`
	CreatedAt     time.Time      `json:"created_at"`
}

// Problem (RFC 7807)

type Problem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}
