package service

import "fmt"

// OJK tariff rates per SE-06/2013 (annualised percentage of IDV).
// Zone 1 = Sumatera; Zone 2 = DKI Jakarta; Zone 3 = Jawa excl. DKI;
// Zone 4 = Kalimantan/Sulawesi; Zone 5 = Maluku/Papua/NTT/NTB.
var ojkRates = map[string]map[string]struct{ Min, Max float64 }{
	"MOTOR_COMPREHENSIVE": {
		"ZONE_1": {1.80, 2.65},
		"ZONE_2": {3.82, 4.20},
		"ZONE_3": {2.53, 3.08},
		"ZONE_4": {2.53, 3.08},
		"ZONE_5": {2.53, 3.08},
	},
	"MOTOR_THIRD_PARTY": {
		"ZONE_1": {0.47, 0.56},
		"ZONE_2": {0.47, 0.56},
		"ZONE_3": {0.47, 0.56},
		"ZONE_4": {0.47, 0.56},
		"ZONE_5": {0.47, 0.56},
	},
	"MOTOR_FIRE_THEFT": {
		"ZONE_1": {1.40, 2.10},
		"ZONE_2": {2.50, 3.00},
		"ZONE_3": {1.80, 2.50},
		"ZONE_4": {1.80, 2.50},
		"ZONE_5": {1.80, 2.50},
	},
}

const (
	adminFeeIDR  int64 = 150_000
	stampDutyIDR int64 = 10_000
)

type PremiumBreakup struct {
	BasePremiumIDR int64
	AdminFeeIDR    int64
	StampDutyIDR   int64
	TotalIDR       int64
	RateUsed       float64
}

// CalcPremium returns the OJK-compliant premium for the given product type, zone, and IDV.
// It uses the midpoint rate if the rate range is valid; returns an error if inputs are unknown.
func CalcPremium(productType, tariffZone string, idvIDR int64) (PremiumBreakup, error) {
	zoneMap, ok := ojkRates[productType]
	if !ok {
		return PremiumBreakup{}, fmt.Errorf("unknown product type: %s", productType)
	}
	rates, ok := zoneMap[tariffZone]
	if !ok {
		return PremiumBreakup{}, fmt.Errorf("unknown tariff zone: %s", tariffZone)
	}
	midRate := (rates.Min + rates.Max) / 2
	basePremium := int64(float64(idvIDR) * midRate / 100.0)
	total := basePremium + adminFeeIDR + stampDutyIDR
	return PremiumBreakup{
		BasePremiumIDR: basePremium,
		AdminFeeIDR:    adminFeeIDR,
		StampDutyIDR:   stampDutyIDR,
		TotalIDR:       total,
		RateUsed:       midRate,
	}, nil
}
