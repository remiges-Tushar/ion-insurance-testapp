package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogService struct {
	db            *pgxpool.Pool
	onixBPPCaller string
}

func NewCatalogService(db *pgxpool.Pool) *CatalogService {
	url := os.Getenv("BPP_ONIX_BPP_CALLER_URL")
	if url == "" {
		url = "http://localhost:8082/bpp/caller"
	}
	return &CatalogService{db: db, onixBPPCaller: url}
}

// ─── Providers ────────────────────────────────────────────────────────────────

type ProviderRow struct {
	ID                 int64          `json:"id"`
	BppID              string         `json:"bpp_id"`
	Name               string         `json:"name"`
	Descriptor         map[string]any `json:"descriptor"`
	Locations          []any          `json:"locations"`
	ProviderAttributes map[string]any `json:"provider_attributes"`
	CreatedAt          time.Time      `json:"created_at"`
}

func (s *CatalogService) CreateProvider(ctx context.Context, bppID, name string, descriptor map[string]any, locations []any, attrs map[string]any) (int64, error) {
	descJSON, _ := json.Marshal(descriptor)
	locJSON, _ := json.Marshal(locations)
	attrsJSON, _ := json.Marshal(attrs)
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO providers (bpp_id, name, descriptor, locations, provider_attributes) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		bppID, name, descJSON, locJSON, attrsJSON,
	).Scan(&id)
	return id, err
}

func (s *CatalogService) ListProviders(ctx context.Context) ([]ProviderRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, bpp_id, name, descriptor, locations, provider_attributes, created_at FROM providers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ProviderRow
	for rows.Next() {
		var r ProviderRow
		var descJSON, locJSON, attrsJSON []byte
		if err := rows.Scan(&r.ID, &r.BppID, &r.Name, &descJSON, &locJSON, &attrsJSON, &r.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(descJSON, &r.Descriptor)
		json.Unmarshal(locJSON, &r.Locations)
		json.Unmarshal(attrsJSON, &r.ProviderAttributes)
		results = append(results, r)
	}
	return results, rows.Err()
}

// ─── Catalogs ─────────────────────────────────────────────────────────────────

func (s *CatalogService) CreateCatalog(ctx context.Context, bppID, name string, descriptor, validity map[string]any, version string, providerID *int64) (int64, error) {
	descJSON, _ := json.Marshal(descriptor)
	validJSON, _ := json.Marshal(validity)
	v := version
	if v == "" {
		v = "1.0.0"
	}
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO catalogs (bpp_id, name, descriptor, validity, version, provider_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		bppID, name, descJSON, validJSON, v, providerID,
	).Scan(&id)
	return id, err
}

func (s *CatalogService) CreateResource(ctx context.Context, bppID, productType, vehicleType, ojkCode string, attrs map[string]any) (int64, error) {
	attrsJSON, _ := json.Marshal(attrs)
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO resources (bpp_id, product_type, vehicle_type, ojk_product_code, resource_attributes) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		bppID, productType, vehicleType, ojkCode, attrsJSON,
	).Scan(&id)
	return id, err
}

func (s *CatalogService) CreateOffer(ctx context.Context, resourceID int64, bppID, zone string, minRate, maxRate float64, attrs map[string]any, validUntil *time.Time) (int64, error) {
	attrsJSON, _ := json.Marshal(attrs)
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO offers (resource_id, bpp_id, tariff_zone, premium_rate_min, premium_rate_max, offer_attributes, valid_until) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		resourceID, bppID, zone, minRate, maxRate, attrsJSON, validUntil,
	).Scan(&id)
	return id, err
}

type CatalogRow struct {
	ID           int64     `json:"id"`
	BppID        string    `json:"bpp_id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	CdsStatus    string    `json:"cds_status"`
	ProviderID   *int64    `json:"provider_id,omitempty"`
	ProviderName string    `json:"provider_name,omitempty"`
}

func (s *CatalogService) ListCatalogs(ctx context.Context) ([]CatalogRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.bpp_id, c.name, c.version, c.created_at,
		       COALESCE((SELECT cpr.cds_status FROM catalog_publish_results cpr WHERE cpr.catalog_id = c.id ORDER BY cpr.id DESC LIMIT 1), 'UNPUBLISHED'),
		       c.provider_id, COALESCE(p.name, '')
		FROM catalogs c
		LEFT JOIN providers p ON p.id = c.provider_id
		ORDER BY c.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []CatalogRow
	for rows.Next() {
		var r CatalogRow
		if err := rows.Scan(&r.ID, &r.BppID, &r.Name, &r.Version, &r.CreatedAt, &r.CdsStatus, &r.ProviderID, &r.ProviderName); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

type ResourceRow struct {
	ID                 int64          `json:"id"`
	BppID              string         `json:"bpp_id"`
	ProductType        string         `json:"product_type"`
	VehicleType        string         `json:"vehicle_type"`
	OJKProductCode     string         `json:"ojk_product_code"`
	ResourceAttributes map[string]any `json:"resource_attributes"`
	CreatedAt          time.Time      `json:"created_at"`
	// Offer fields (populated by discover query via LEFT JOIN)
	TariffZone     string  `json:"tariff_zone"`
	PremiumRateMin float64 `json:"premium_rate_min"`
	PremiumRateMax float64 `json:"premium_rate_max"`
}

func (s *CatalogService) ListResources(ctx context.Context) ([]ResourceRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, bpp_id, product_type, vehicle_type, ojk_product_code, resource_attributes, created_at FROM resources ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ResourceRow
	for rows.Next() {
		var r ResourceRow
		var attrsJSON []byte
		if err := rows.Scan(&r.ID, &r.BppID, &r.ProductType, &r.VehicleType, &r.OJKProductCode, &attrsJSON, &r.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(attrsJSON, &r.ResourceAttributes)
		results = append(results, r)
	}
	return results, rows.Err()
}

type OfferRow struct {
	ID              int64          `json:"id"`
	ResourceID      int64          `json:"resource_id"`
	ResourceName    string         `json:"resource_name,omitempty"`
	BppID           string         `json:"bpp_id"`
	TariffZone      string         `json:"tariff_zone"`
	PremiumRateMin  float64        `json:"premium_rate_min"`
	PremiumRateMax  float64        `json:"premium_rate_max"`
	OfferAttributes map[string]any `json:"offer_attributes"`
	ValidUntil      *time.Time     `json:"valid_until"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (s *CatalogService) ListOffers(ctx context.Context) ([]OfferRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT o.id, o.resource_id, o.bpp_id, o.tariff_zone, o.premium_rate_min, o.premium_rate_max,
		       o.offer_attributes, o.valid_until, o.created_at,
		       COALESCE((r.resource_attributes->>'descriptor')::text, r.ojk_product_code, '')
		FROM offers o
		LEFT JOIN resources r ON r.id = o.resource_id
		ORDER BY o.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []OfferRow
	for rows.Next() {
		var r OfferRow
		var attrsJSON []byte
		var rawDesc string
		if err := rows.Scan(&r.ID, &r.ResourceID, &r.BppID, &r.TariffZone, &r.PremiumRateMin, &r.PremiumRateMax, &attrsJSON, &r.ValidUntil, &r.CreatedAt, &rawDesc); err != nil {
			return nil, err
		}
		json.Unmarshal(attrsJSON, &r.OfferAttributes)
		// Extract resource descriptor name from JSONB
		var desc map[string]any
		if json.Unmarshal([]byte(rawDesc), &desc) == nil {
			if name, ok := desc["name"].(string); ok {
				r.ResourceName = name
			}
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *CatalogService) logMessage(ctx context.Context, action, txnID string, payload map[string]any) {
	payloadJSON, _ := json.Marshal(payload)
	s.db.Exec(ctx,
		`INSERT INTO beckn_message_log (action, transaction_id, payload) VALUES ($1,$2,$3)`,
		action, txnID, payloadJSON,
	)
}

// PublishCatalog sends a Beckn v2 catalog/publish message through beckn-onix to
// the Cataloging Service (CS). The CS validates and calls back with
// catalog/on_publish which updates the status from PENDING to ACCEPTED/REJECTED.
func (s *CatalogService) PublishCatalog(ctx context.Context, catalogID int64, bppID string) error {
	// Fetch catalog + its linked provider row
	var catalogName, catalogVersion string
	var catalogDescJSON []byte
	var providerID *int64
	err := s.db.QueryRow(ctx,
		`SELECT name, version, descriptor, provider_id FROM catalogs WHERE id = $1`, catalogID,
	).Scan(&catalogName, &catalogVersion, &catalogDescJSON, &providerID)
	if err != nil {
		return fmt.Errorf("fetch catalog %d: %w", catalogID, err)
	}

	// Build provider block — prefer the linked providers row; fall back to provider_accounts
	providerBlock := map[string]any{
		"id":         bppID,
		"descriptor": map[string]any{"name": bppID},
	}
	if providerID != nil {
		var pName string
		var pDescJSON, pLocJSON, pAttrsJSON []byte
		if scanErr := s.db.QueryRow(ctx,
			`SELECT name, descriptor, locations, provider_attributes FROM providers WHERE id = $1`, *providerID,
		).Scan(&pName, &pDescJSON, &pLocJSON, &pAttrsJSON); scanErr == nil {
			var pDesc, pAttrs map[string]any
			var pLoc []any
			json.Unmarshal(pDescJSON, &pDesc)
			json.Unmarshal(pLocJSON, &pLoc)
			json.Unmarshal(pAttrsJSON, &pAttrs)
			if pDesc == nil {
				pDesc = map[string]any{"name": pName}
			}
			// Attributes requires @context + @type (additionalProperties: true so other fields are fine)
			if pAttrs == nil {
				pAttrs = map[string]any{}
			}
			if _, ok := pAttrs["@context"]; !ok {
				pAttrs["@context"] = "https://schema.ion.id/core/identity/v1/context.jsonld"
			}
			if _, ok := pAttrs["@type"]; !ok {
				pAttrs["@type"] = "ion:BusinessIdentity"
			}
			cleanLoc := sanitizeLocations(pLoc)
			pb := map[string]any{
				"id":                 fmt.Sprintf("PROV-%d", *providerID),
				"descriptor":         pDesc,
				"providerAttributes": pAttrs,
			}
			if len(cleanLoc) > 0 {
				pb["availableAt"] = cleanLoc
			}
			providerBlock = pb
		}
	} else {
		// Fall back to provider_accounts company name
		var companyName string
		_ = s.db.QueryRow(ctx,
			`SELECT company_name FROM provider_accounts ORDER BY id LIMIT 1`,
		).Scan(&companyName)
		if companyName != "" {
			providerBlock["descriptor"] = map[string]any{"name": companyName}
		}
	}

	// Fetch full resource details for this BPP — Beckn v2 Resource objects
	resRows, err := s.db.Query(ctx,
		`SELECT id, product_type, vehicle_type, ojk_product_code, resource_attributes
		 FROM resources WHERE bpp_id = $1 ORDER BY id`, bppID,
	)
	if err != nil {
		return fmt.Errorf("fetch resources: %w", err)
	}
	var resources []map[string]any
	for resRows.Next() {
		var rid int64
		var productType, vehicleType, ojkCode string
		var attrsJSON []byte
		if scanErr := resRows.Scan(&rid, &productType, &vehicleType, &ojkCode, &attrsJSON); scanErr != nil {
			continue
		}
		var attrs map[string]any
		json.Unmarshal(attrsJSON, &attrs)

		// Extract human-readable descriptor from stored resource_attributes
		resDescriptor := map[string]any{
			"name": fmt.Sprintf("%s — %s", productType, vehicleType),
		}
		if attrs != nil {
			if d, ok := attrs["descriptor"].(map[string]any); ok {
				resDescriptor = d
			}
		}

		// Fetch linked offers for this resource
		offerRows, _ := s.db.Query(ctx,
			`SELECT id, tariff_zone, premium_rate_min, premium_rate_max, offer_attributes
			 FROM offers WHERE resource_id = $1 ORDER BY id`, rid,
		)
		var linkedOffers []map[string]any
		if offerRows != nil {
			for offerRows.Next() {
				var oid int64
				var zone string
				var rateMin, rateMax float64
				var oAttrsJSON []byte
				if scanErr := offerRows.Scan(&oid, &zone, &rateMin, &rateMax, &oAttrsJSON); scanErr != nil {
					continue
				}
				var oAttrs map[string]any
				json.Unmarshal(oAttrsJSON, &oAttrs)
				linkedOffers = append(linkedOffers, map[string]any{
					"id":             fmt.Sprintf("OFFER-%d", oid),
					"tariffZone":     zone,
					"premiumRateMin": rateMin,
					"premiumRateMax": rateMax,
					"offerAttributes": oAttrs,
				})
			}
			offerRows.Close()
		}

		// Merge productType/vehicleType/ojkProductCode into resourceAttributes
		// so they stay inside the extension point (top-level Resource schema
		// has additionalProperties:false — only id/descriptor/resourceAttributes allowed).
		if attrs == nil {
			attrs = map[string]any{}
		}
		if _, exists := attrs["productType"]; !exists {
			attrs["productType"] = productType
		}
		if _, exists := attrs["vehicleType"]; !exists {
			attrs["vehicleType"] = vehicleType
		}
		if _, exists := attrs["ojkProductCode"]; !exists {
			attrs["ojkProductCode"] = ojkCode
		}
		res := map[string]any{
			"id":                 fmt.Sprintf("RES-%d", rid),
			"descriptor":         resDescriptor,
			"resourceAttributes": attrs,
		}
		if len(linkedOffers) > 0 {
			res["offers"] = linkedOffers
		}
		resources = append(resources, res)
	}
	resRows.Close()

	// Beckn v2 Catalog anyOf requires at least resources or offers.
	if len(resources) == 0 {
		resources = []map[string]any{
			{
				"id":         fmt.Sprintf("CAT-%d-default", catalogID),
				"descriptor": map[string]any{"name": catalogName},
			},
		}
	}

	txnID := fmt.Sprintf("pub-%d-%d", catalogID, time.Now().UnixMilli())

	// Beckn v2 catalog/publish message.
	// Catalog.id uses "CAT-{n}" so the CS mock can parse the numeric ID and
	// echo it back in catalog/on_publish for DB row lookup.
	// Field names are camelCase per the Beckn v2 spec (additionalProperties: false).
	payload := map[string]any{
		"context": map[string]any{
			"action":         "catalog/publish",
			"version":        "2.0.0",
			"domain":         "ion:finance",
			"bpp_id":         bppID,
			"transaction_id": txnID,
			"message_id":     fmt.Sprintf("msg-%d-%d", catalogID, time.Now().UnixNano()),
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
			"ttl":            "PT30S",
		},
		"message": map[string]any{
			"catalogs": []map[string]any{
				{
					"id": fmt.Sprintf("CAT-%d", catalogID),
					"descriptor": map[string]any{
						"name": catalogName,
					},
					"bppId":     bppID,
					"provider":  providerBlock,
					"resources": resources,
				},
			},
		},
	}
	s.logMessage(ctx, "catalog/publish", txnID, payload)
	body, _ := json.Marshal(payload)

	// POST to beckn-onix BPP caller — it signs the request and forwards to CS
	resp, err := http.Post(s.onixBPPCaller+"/catalog/publish", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("forward catalog/publish to onix-bpp caller: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onix-bpp caller returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Insert PENDING; on_publish callback from CS will update to ACCEPTED/REJECTED
	_, updateErr := s.db.Exec(ctx,
		`INSERT INTO catalog_publish_results (catalog_id, cds_status) VALUES ($1, 'PENDING')`,
		catalogID,
	)
	return updateErr
}

// sanitizeLocations converts stored frontend location objects into valid Beckn v2 Location objects.
// Beckn v2 Location: { geo (required, GeoJSON), address (optional, Address) }
// Address valid fields: addressCountry, addressLocality, addressRegion, extendedAddress, postalCode, streetAddress
func sanitizeLocations(locs []any) []any {
	validAddr := map[string]bool{
		"addressCountry": true, "addressLocality": true, "addressRegion": true,
		"extendedAddress": true, "postalCode": true, "streetAddress": true,
	}
	// legacy field mapping from old frontend form names
	addrMap := map[string]string{
		"city": "addressLocality", "state": "addressRegion",
		"country": "addressCountry", "areaCode": "postalCode",
		"street": "streetAddress",
	}
	out := make([]any, 0, len(locs))
	for _, l := range locs {
		loc, ok := l.(map[string]any)
		if !ok {
			continue
		}
		clean := map[string]any{}

		// geo: prefer stored GeoJSON map, else parse "lat,lng" gps string
		if geo, ok := loc["geo"]; ok {
			clean["geo"] = geo
		} else if gps, ok := loc["gps"].(string); ok && gps != "" {
			var lat, lng float64
			if n, err := fmt.Sscanf(gps, "%f,%f", &lat, &lng); n == 2 && err == nil {
				clean["geo"] = map[string]any{
					"type":        "Point",
					"coordinates": []float64{lng, lat}, // GeoJSON: [longitude, latitude]
				}
			}
		}
		if _, hasGeo := clean["geo"]; !hasGeo {
			continue // Location.geo is required — skip locations without it
		}

		// address: map legacy fields + pass through already-valid fields
		if rawAddr, ok := loc["address"].(map[string]any); ok {
			cleanAddr := map[string]any{}
			// build extendedAddress from door+building if present
			door, _ := rawAddr["door"].(string)
			building, _ := rawAddr["building"].(string)
			if door != "" || building != "" {
				parts := ""
				if door != "" {
					parts = door
				}
				if building != "" {
					if parts != "" {
						parts += ", " + building
					} else {
						parts = building
					}
				}
				cleanAddr["extendedAddress"] = parts
			}
			for k, v := range rawAddr {
				if validAddr[k] {
					cleanAddr[k] = v
				} else if mapped, ok := addrMap[k]; ok {
					cleanAddr[mapped] = v
				}
			}
			if len(cleanAddr) > 0 {
				clean["address"] = cleanAddr
			}
		}
		out = append(out, clean)
	}
	return out
}

func (s *CatalogService) HandleOnPublish(ctx context.Context, catalogID int64, status string, payload map[string]any) error {
	txnID := fmt.Sprintf("onpub-%d", catalogID)
	if pctx, ok := payload["context"].(map[string]any); ok {
		if v, ok := pctx["transaction_id"].(string); ok && v != "" {
			txnID = v
		}
	}
	s.logMessage(ctx, "catalog/on_publish", txnID, payload)

	payloadJSON, _ := json.Marshal(payload)
	now := time.Now()
	_, err := s.db.Exec(ctx,
		`UPDATE catalog_publish_results SET cds_status=$1, result_payload=$2, published_at=$3 WHERE catalog_id=$4`,
		status, payloadJSON, now, catalogID,
	)
	return err
}
