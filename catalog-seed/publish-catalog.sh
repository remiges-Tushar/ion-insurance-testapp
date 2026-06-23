#!/bin/bash
# Seed script: register insurer, create catalog, resources, offers, and publish
set -e

BPP_URL="${BPP_URL:-http://localhost:8080}"
EMAIL="${SEED_EMAIL:-admin@ptasuransimaju.co.id}"
PASSWORD="${SEED_PASSWORD:-SecurePass123}"
COMPANY="${SEED_COMPANY:-PT Asuransi Maju}"
OJK_LICENSE="${SEED_OJK:-OJK-KEP-234}"

echo "==> Registering insurer..."
REG_RESP=$(curl -s -X POST "$BPP_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"company_name\":\"$COMPANY\",\"ojk_license\":\"$OJK_LICENSE\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" || true)
echo "Register: $REG_RESP"

echo "==> Logging in..."
LOGIN_RESP=$(curl -s -X POST "$BPP_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "Login: $LOGIN_RESP"

TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
  echo "ERROR: Could not extract token from login response"
  exit 1
fi
echo "==> Got token: ${TOKEN:0:20}..."

AUTH_HEADER="Authorization: Bearer $TOKEN"

echo "==> Creating catalog..."
CAT_RESP=$(curl -s -X POST "$BPP_URL/api/v1/catalogs" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{"name":"PT Asuransi Maju Motor Insurance Catalog","version":"1.0.0","descriptor":{"name":"PT Asuransi Maju"},"validity":{}}')
echo "Catalog: $CAT_RESP"
CATALOG_ID=$(echo "$CAT_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "==> Catalog ID: $CATALOG_ID"

echo "==> Creating MOTOR_COMPREHENSIVE resource..."
RES1_RESP=$(curl -s -X POST "$BPP_URL/api/v1/resources" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "product_type": "MOTOR_COMPREHENSIVE",
    "vehicle_type": "FOUR_WHEELER",
    "ojk_product_code": "AMAJU-KOMPREHENSIF-4W-001",
    "resource_attributes": {
      "@type": "ion:InsuranceProduct",
      "productType": "MOTOR_COMPREHENSIVE",
      "vehicleType": "FOUR_WHEELER",
      "coverageInclusions": ["PARTIAL_LOSS","TOTAL_LOSS","THEFT","FIRE","THIRD_PARTY_LIABILITY","NATURAL_DISASTER","ROADSIDE_ASSISTANCE"]
    }
  }')
echo "Resource 1: $RES1_RESP"
RES1_ID=$(echo "$RES1_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "==> Resource 1 ID: $RES1_ID"

echo "==> Creating MOTOR_THIRD_PARTY resource..."
RES2_RESP=$(curl -s -X POST "$BPP_URL/api/v1/resources" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "product_type": "MOTOR_THIRD_PARTY",
    "vehicle_type": "FOUR_WHEELER",
    "ojk_product_code": "AMAJU-TLO-4W-001",
    "resource_attributes": {
      "@type": "ion:InsuranceProduct",
      "productType": "MOTOR_THIRD_PARTY",
      "vehicleType": "FOUR_WHEELER",
      "coverageInclusions": ["TOTAL_LOSS","THEFT","FIRE","THIRD_PARTY_LIABILITY"]
    }
  }')
echo "Resource 2: $RES2_RESP"
RES2_ID=$(echo "$RES2_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "==> Resource 2 ID: $RES2_ID"

echo "==> Creating Zone 3 offer for MOTOR_COMPREHENSIVE..."
OFFER1_RESP=$(curl -s -X POST "$BPP_URL/api/v1/offers" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{
    \"resource_id\": $RES1_ID,
    \"tariff_zone\": \"ZONE_3\",
    \"premium_rate_min\": 2.53,
    \"premium_rate_max\": 3.08,
    \"offer_attributes\": {\"@type\": \"ion:PolicyQuote\", \"tariffZone\": \"ZONE_3\", \"deductibleIDR\": 300000}
  }")
echo "Offer 1: $OFFER1_RESP"

echo "==> Creating Zone 3 offer for MOTOR_THIRD_PARTY..."
OFFER2_RESP=$(curl -s -X POST "$BPP_URL/api/v1/offers" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{
    \"resource_id\": $RES2_ID,
    \"tariff_zone\": \"ZONE_3\",
    \"premium_rate_min\": 0.47,
    \"premium_rate_max\": 0.56,
    \"offer_attributes\": {\"@type\": \"ion:PolicyQuote\", \"tariffZone\": \"ZONE_3\", \"deductibleIDR\": 200000}
  }")
echo "Offer 2: $OFFER2_RESP"

echo "==> Publishing catalog..."
PUB_RESP=$(curl -s -X POST "$BPP_URL/api/v1/catalog/publish" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{\"catalog_id\": $CATALOG_ID}")
echo "Publish: $PUB_RESP"

echo ""
echo "==> Catalog seed complete!"
echo "    BPP Frontend: http://localhost:3001 (login: $EMAIL / $PASSWORD)"
echo "    BAP Frontend: http://localhost:3000 (no login needed)"
