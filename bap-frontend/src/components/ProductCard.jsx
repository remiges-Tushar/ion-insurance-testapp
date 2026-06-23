import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Shield, ChevronDown, ChevronUp, Tag, Percent, FileText } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

// ─── Shared display atoms ──────────────────────────────────────────────────────

const INCLUSION_COLORS = {
  PARTIAL_LOSS:               'bg-blue-50 text-blue-700 border-blue-200',
  TOTAL_LOSS:                 'bg-indigo-50 text-indigo-700 border-indigo-200',
  THEFT:                      'bg-purple-50 text-purple-700 border-purple-200',
  FIRE:                       'bg-orange-50 text-orange-700 border-orange-200',
  THIRD_PARTY_LIABILITY:      'bg-teal-50 text-teal-700 border-teal-200',
  NATURAL_DISASTER:           'bg-cyan-50 text-cyan-700 border-cyan-200',
  ROADSIDE_ASSISTANCE:        'bg-green-50 text-green-700 border-green-200',
  PERSONAL_ACCIDENT_DRIVER:   'bg-rose-50 text-rose-700 border-rose-200',
  PERSONAL_ACCIDENT_PASSENGER:'bg-pink-50 text-pink-700 border-pink-200',
}

function InclusionChip({ code }) {
  const cls = INCLUSION_COLORS[code] || 'bg-slate-50 text-slate-600 border-slate-200'
  const label = code.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase())
  return (
    <span className={`inline-flex items-center border text-xs font-medium px-2 py-0.5 rounded-full ${cls}`}>
      {label}
    </span>
  )
}

function ExclusionChip({ code }) {
  const label = code.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase())
  return (
    <span className="inline-flex items-center border border-red-100 bg-red-50 text-red-600 text-xs px-2 py-0.5 rounded-full">
      {label}
    </span>
  )
}

const TYPE_MAP = {
  MOTOR_COMPREHENSIVE: { label: 'Comprehensive', cls: 'bg-emerald-100 text-emerald-800' },
  MOTOR_THIRD_PARTY:   { label: 'Third Party',   cls: 'bg-amber-100 text-amber-800' },
  MOTOR_FIRE_THEFT:    { label: 'Fire & Theft',  cls: 'bg-orange-100 text-orange-800' },
  TWO_WHEELER:         { label: 'Two Wheeler',   cls: 'bg-violet-100 text-violet-800' },
  FOUR_WHEELER:        { label: 'Four Wheeler',  cls: 'bg-blue-100 text-blue-800' },
  COMMERCIAL_VEHICLE:  { label: 'Commercial',    cls: 'bg-slate-100 text-slate-700' },
}

function TypeBadge({ value }) {
  const { label, cls } = TYPE_MAP[value] || { label: value, cls: 'bg-slate-100 text-slate-600' }
  return <span className={`text-xs font-semibold px-2.5 py-1 rounded-lg ${cls}`}>{label}</span>
}

const ZONE_COLORS = {
  ZONE_1: 'bg-red-100 text-red-800',
  ZONE_2: 'bg-orange-100 text-orange-800',
  ZONE_3: 'bg-yellow-100 text-yellow-800',
  ZONE_4: 'bg-lime-100 text-lime-800',
  ZONE_5: 'bg-green-100 text-green-800',
}

function ZoneBadge({ zone }) {
  if (!zone) return null
  return (
    <span className={`text-xs font-bold px-2.5 py-1 rounded-lg ${ZONE_COLORS[zone] || 'bg-slate-100 text-slate-700'}`}>
      {zone.replace('_', ' ')}
    </span>
  )
}

function NcdLadder({ discounts }) {
  if (!discounts?.length) return null
  const sorted = [...discounts].sort((a, b) => Number(a.discountPercent) - Number(b.discountPercent))
  function ncdLabel(type) {
    const m = type?.match(/NCD_(\d+)_YEAR(S)?(_PLUS)?/)
    if (!m) return type?.replace(/_/g, ' ') || type
    return m[3] ? `${m[1]}+ yr` : `${m[1]} yr`
  }
  return (
    <div className="flex items-center gap-0 flex-wrap">
      {sorted.map((d, i) => (
        <React.Fragment key={i}>
          <div className="flex flex-col items-center">
            <span className="text-xs font-bold text-emerald-700">{Number(d.discountPercent)}%</span>
            <span className="text-xs text-slate-500 leading-tight">{ncdLabel(d.type)}</span>
          </div>
          {i < sorted.length - 1 && <div className="w-5 h-px bg-emerald-200 mx-1 self-center mb-3" />}
        </React.Fragment>
      ))}
    </div>
  )
}

function AddOnRow({ addon }) {
  return (
    <div className="flex items-center justify-between py-1.5 border-b border-slate-50 last:border-0">
      <div className="flex items-center gap-2 min-w-0">
        <span className="w-1.5 h-1.5 rounded-full bg-blue-400 shrink-0" />
        <div className="min-w-0">
          <span className="text-sm text-slate-700">{addon.name}</span>
          <span className="text-xs text-slate-400 ml-1.5">({addon.code})</span>
        </div>
      </div>
      <span className="text-sm font-semibold text-blue-700 shrink-0 ml-3">+{Number(addon.additionalPremiumRate)}%</span>
    </div>
  )
}

// ─── ProductCard ───────────────────────────────────────────────────────────────

export default function ProductCard({ product, onGetQuote }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  const ra = product.resourceAttributes || product
  const name = product.descriptor?.name || ra.descriptor?.name || product.insurer_name || 'Insurance Product'
  const shortDesc = product.descriptor?.shortDesc || ra.descriptor?.shortDesc
  const insurer = product.insurer_name || product.insurerName || ''
  const productType = product.productType || product.product_type || ra.productType || ''
  const vehicleType = product.vehicleType || product.vehicle_type || ra.vehicleType || ''
  const tariffZone = product.tariffZone || product.tariff_zone || ra.tariffZone || ''
  const rateMin = product.premiumRateRange?.min ?? product.premium_rate_min ?? ra.premiumRateRange?.min ?? 0
  const rateMax = product.premiumRateRange?.max ?? product.premium_rate_max ?? ra.premiumRateRange?.max ?? 0
  const ojkCode = product.ojkProductCode || ra.ojkProductCode
  const ojkLicense = product.ojkLicenseNumber || ra.ojkLicenseNumber
  const deductible = product.deductibleAmount || ra.deductibleAmount
  const inclusions = product.coverageInclusions || ra.coverageInclusions || []
  const exclusions = product.standardExclusions || ra.standardExclusions || []
  const addOns = product.addOnOptions || ra.addOnOptions || []
  const discounts = product.discounts || ra.discounts || []

  const hasExpandable = addOns.length > 0 || discounts.length > 0 || exclusions.length > 0

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white border border-slate-200 rounded-2xl overflow-hidden shadow-sm hover:shadow-md transition-shadow flex flex-col"
    >
      {/* Header */}
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            {insurer && <p className="text-xs text-slate-400 font-medium mb-0.5 truncate">{insurer}</p>}
            <h3 className="text-base font-semibold text-slate-900 leading-tight">{name}</h3>
            {shortDesc && <p className="text-xs text-slate-500 mt-1 line-clamp-2">{shortDesc}</p>}
          </div>
          <div className="p-2 bg-blue-50 rounded-xl shrink-0">
            <Shield className="text-blue-600" size={20} />
          </div>
        </div>

        {/* Type + Zone badges */}
        <div className="flex flex-wrap gap-1.5 mt-3">
          {productType && <TypeBadge value={productType} />}
          {vehicleType && <TypeBadge value={vehicleType} />}
          {tariffZone && <ZoneBadge zone={tariffZone} />}
        </div>

        {/* Premium rate + deductible */}
        <div className="flex items-center gap-3 mt-3 flex-wrap">
          {(rateMin || rateMax) > 0 && (
            <div className="flex items-center gap-1 text-sm">
              <Percent size={13} className="text-slate-400" />
              <span className="font-semibold text-slate-800">{rateMin}% – {rateMax}%</span>
              <span className="text-slate-400 text-xs">{t('product.per_year')}</span>
            </div>
          )}
          {deductible && (
            <span className="text-xs text-slate-500 bg-slate-50 border border-slate-200 px-2 py-0.5 rounded-lg">
              Deductible: <span className="font-semibold text-slate-700">IDR {Number(deductible).toLocaleString('id-ID')}</span>
            </span>
          )}
        </div>

        {/* OJK info */}
        {(ojkCode || ojkLicense) && (
          <div className="flex gap-3 mt-2 flex-wrap">
            {ojkCode && <span className="text-xs text-slate-400">OJK Code: <span className="font-mono text-slate-600">{ojkCode}</span></span>}
            {ojkLicense && <span className="text-xs text-slate-400">License: <span className="font-mono text-slate-600">{ojkLicense}</span></span>}
          </div>
        )}
      </div>

      {/* Coverage inclusions — always visible */}
      {inclusions.length > 0 && (
        <div className="px-5 py-3 border-t border-slate-50 bg-slate-50/50">
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
            <Shield size={11} /> Coverage
          </p>
          <div className="flex flex-wrap gap-1">
            {inclusions.map(c => <InclusionChip key={c} code={c} />)}
          </div>
        </div>
      )}

      {/* Expand toggle */}
      {hasExpandable && (
        <button type="button" onClick={() => setExpanded(v => !v)}
          className="w-full flex items-center justify-between px-5 py-2 border-t border-slate-100 text-xs font-semibold text-slate-500 hover:bg-slate-50 transition-colors">
          <span className="flex items-center gap-3">
            {addOns.length > 0 && <span className="flex items-center gap-1"><Tag size={11} className="text-blue-500" />{addOns.length} Add-On{addOns.length > 1 ? 's' : ''}</span>}
            {discounts.length > 0 && <span className="flex items-center gap-1"><Percent size={11} className="text-emerald-500" />{discounts.length} NCD Level{discounts.length > 1 ? 's' : ''}</span>}
            {exclusions.length > 0 && <span className="flex items-center gap-1"><FileText size={11} className="text-red-400" />{exclusions.length} Exclusion{exclusions.length > 1 ? 's' : ''}</span>}
          </span>
          {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </button>
      )}

      {/* Expanded detail */}
      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.18 }}
            className="overflow-hidden border-t border-slate-100">

            {addOns.length > 0 && (
              <div className="px-5 py-3 border-b border-slate-50">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <Tag size={11} className="text-blue-500" /> Optional Add-Ons
                </p>
                {addOns.map((a, i) => <AddOnRow key={i} addon={a} />)}
              </div>
            )}

            {discounts.length > 0 && (
              <div className="px-5 py-3 border-b border-slate-50">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <Percent size={11} className="text-emerald-500" /> No-Claims Discount
                </p>
                <NcdLadder discounts={discounts} />
                <p className="text-xs text-slate-400 mt-1.5">Earn more discount each claim-free year.</p>
              </div>
            )}

            {exclusions.length > 0 && (
              <div className="px-5 py-3">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <FileText size={11} className="text-red-400" /> Standard Exclusions
                </p>
                <div className="flex flex-wrap gap-1">
                  {exclusions.map(e => <ExclusionChip key={e} code={e} />)}
                </div>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>

      {/* CTA */}
      <div className="px-5 pb-5 pt-3 mt-auto border-t border-slate-50">
        <button
          onClick={() => onGetQuote(product)}
          className="w-full bg-blue-600 text-white py-2.5 rounded-xl font-semibold hover:bg-blue-700 transition-colors text-sm"
        >
          {t('product.get_quote')}
        </button>
      </div>
    </motion.div>
  )
}
