import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronDown, ChevronUp, Shield, Tag, Percent, FileText, Link, Building2, Box, Layers } from 'lucide-react'
import client from '../api/client.js'
import StatusBadge from '../components/StatusBadge.jsx'
import { fmtDate, fmtDateTime } from '../utils/date.js'

// ─── Shared tab bar ───────────────────────────────────────────────────────────

function Tab({ active, onClick, label, count }) {
  return (
    <button onClick={onClick}
      className={`flex items-center gap-2 px-5 py-2.5 text-sm font-medium rounded-xl transition-all ${active ? 'bg-blue-600 text-white shadow-sm' : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'}`}>
      {label}
      {count != null && (
        <span className={`text-xs px-1.5 py-0.5 rounded-full font-semibold ${active ? 'bg-blue-500 text-white' : 'bg-slate-100 text-slate-500'}`}>{count}</span>
      )}
    </button>
  )
}

// ─── Products tab — rich cards ────────────────────────────────────────────────

const INCLUSION_COLORS = {
  PARTIAL_LOSS: 'bg-blue-50 text-blue-700 border-blue-200',
  TOTAL_LOSS: 'bg-indigo-50 text-indigo-700 border-indigo-200',
  THEFT: 'bg-purple-50 text-purple-700 border-purple-200',
  FIRE: 'bg-orange-50 text-orange-700 border-orange-200',
  THIRD_PARTY_LIABILITY: 'bg-teal-50 text-teal-700 border-teal-200',
  NATURAL_DISASTER: 'bg-cyan-50 text-cyan-700 border-cyan-200',
  ROADSIDE_ASSISTANCE: 'bg-green-50 text-green-700 border-green-200',
  PERSONAL_ACCIDENT_DRIVER: 'bg-rose-50 text-rose-700 border-rose-200',
  PERSONAL_ACCIDENT_PASSENGER: 'bg-pink-50 text-pink-700 border-pink-200',
}

function InclusionChip({ code }) {
  const cls = INCLUSION_COLORS[code] || 'bg-slate-50 text-slate-600 border-slate-200'
  const label = code.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase())
  return <span className={`inline-flex items-center border text-xs font-medium px-2 py-0.5 rounded-full ${cls}`}>{label}</span>
}

function ExclusionChip({ code }) {
  const label = code.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase())
  return <span className="inline-flex items-center border border-red-100 bg-red-50 text-red-600 text-xs px-2 py-0.5 rounded-full">{label}</span>
}

function TypeBadge({ value }) {
  const map = {
    MOTOR_COMPREHENSIVE: { label: 'Comprehensive', cls: 'bg-emerald-100 text-emerald-800' },
    MOTOR_THIRD_PARTY: { label: 'Third Party', cls: 'bg-amber-100 text-amber-800' },
    MOTOR_FIRE_THEFT: { label: 'Fire & Theft', cls: 'bg-orange-100 text-orange-800' },
    TWO_WHEELER: { label: 'Two Wheeler', cls: 'bg-violet-100 text-violet-800' },
    FOUR_WHEELER: { label: 'Four Wheeler', cls: 'bg-blue-100 text-blue-800' },
    COMMERCIAL_VEHICLE: { label: 'Commercial', cls: 'bg-slate-100 text-slate-700' },
  }
  const { label, cls } = map[value] || { label: value, cls: 'bg-slate-100 text-slate-600' }
  return <span className={`text-xs font-semibold px-2.5 py-1 rounded-lg ${cls}`}>{label}</span>
}

function ZoneBadge({ zone }) {
  const colors = {
    ZONE_1: 'bg-red-100 text-red-800',
    ZONE_2: 'bg-orange-100 text-orange-800',
    ZONE_3: 'bg-yellow-100 text-yellow-800',
    ZONE_4: 'bg-lime-100 text-lime-800',
    ZONE_5: 'bg-green-100 text-green-800',
  }
  return <span className={`text-xs font-bold px-2.5 py-1 rounded-lg ${colors[zone] || 'bg-slate-100 text-slate-700'}`}>{zone?.replace('_', ' ')}</span>
}

function NcdLadder({ discounts }) {
  if (!discounts?.length) return <span className="text-xs text-slate-400">—</span>

  // Sort by discount percent ascending
  const sorted = [...discounts].sort((a, b) => Number(a.discountPercent) - Number(b.discountPercent))

  // Normalize type to a human label
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
          {i < sorted.length - 1 && <div className="w-5 h-px bg-emerald-200 mx-1 mt-0 self-center mb-3" />}
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

function ProductCard({ product }) {
  const [expanded, setExpanded] = useState(false)

  // Normalise field names (snake_case from API vs camelCase)
  const ra = product.resource_attributes || {}
  const name = ra.descriptor?.name || product.product_type || product.productType || `Product #${product.id}`
  const shortDesc = ra.descriptor?.shortDesc || ra.descriptor?.short_desc
  const productType = product.product_type || product.productType || ra.productType
  const vehicleType = product.vehicle_type || product.vehicleType || ra.vehicleType
  const tariffZone = ra.tariffZone || ra.tariff_zone
  const rateMin = ra.premiumRateRange?.min ?? ra.premium_rate_min
  const rateMax = ra.premiumRateRange?.max ?? ra.premium_rate_max
  const ojkCode = product.ojk_product_code || product.ojkProductCode || ra.ojkProductCode
  const ojkLicense = ra.ojkLicenseNumber || ra.ojk_license_number
  const deductible = ra.deductibleAmount || ra.deductible_amount
  const inclusions = ra.coverageInclusions || ra.coverage_inclusions || []
  const exclusions = ra.standardExclusions || ra.standard_exclusions || []
  const addOns = ra.addOnOptions || ra.add_on_options || []
  const discounts = ra.discounts || []

  return (
    <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
      className="bg-white border border-slate-200 rounded-2xl overflow-hidden shadow-sm hover:shadow-md transition-shadow">

      {/* Card header */}
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-base font-semibold text-slate-900 leading-tight">{name}</h3>
            {shortDesc && <p className="text-xs text-slate-500 mt-1 line-clamp-2">{shortDesc}</p>}
          </div>
          <div className="flex flex-col gap-1.5 items-end shrink-0">
            <TypeBadge value={productType} />
            <TypeBadge value={vehicleType} />
          </div>
        </div>

        {/* Key stats row */}
        <div className="flex items-center gap-3 mt-4 flex-wrap">
          {tariffZone && <ZoneBadge zone={tariffZone} />}
          {rateMin != null && rateMax != null && (
            <div className="flex items-center gap-1 text-sm">
              <Percent size={14} className="text-slate-400" />
              <span className="font-semibold text-slate-800">{rateMin}% – {rateMax}%</span>
              <span className="text-slate-400 text-xs">premium rate</span>
            </div>
          )}
          {deductible && (
            <div className="text-xs text-slate-500 bg-slate-50 border border-slate-200 px-2.5 py-1 rounded-lg">
              Deductible: <span className="font-semibold text-slate-700">IDR {Number(deductible).toLocaleString('id-ID')}</span>
            </div>
          )}
        </div>

        {/* OJK info */}
        <div className="flex gap-4 mt-3 flex-wrap">
          {ojkCode && <span className="text-xs text-slate-400">OJK Code: <span className="font-mono text-slate-600">{ojkCode}</span></span>}
          {ojkLicense && <span className="text-xs text-slate-400">License: <span className="font-mono text-slate-600">{ojkLicense}</span></span>}
        </div>
      </div>

      {/* Coverage inclusions (always visible) */}
      {inclusions.length > 0 && (
        <div className="px-5 py-3 border-t border-slate-50 bg-slate-50/50">
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
            <Shield size={12} /> Coverage Inclusions
          </p>
          <div className="flex flex-wrap gap-1.5">
            {inclusions.map(c => <InclusionChip key={c} code={c} />)}
          </div>
        </div>
      )}

      {/* Expand/collapse toggle */}
      {(addOns.length > 0 || discounts.length > 0 || exclusions.length > 0) && (
        <button type="button" onClick={() => setExpanded(v => !v)}
          className="w-full flex items-center justify-between px-5 py-2.5 border-t border-slate-100 text-xs font-semibold text-slate-500 hover:bg-slate-50 transition-colors">
          <span className="flex items-center gap-3">
            {addOns.length > 0 && <span className="flex items-center gap-1"><Tag size={12} className="text-blue-500" />{addOns.length} Add-On{addOns.length > 1 ? 's' : ''}</span>}
            {discounts.length > 0 && <span className="flex items-center gap-1"><Percent size={12} className="text-emerald-500" />{discounts.length} NCD Level{discounts.length > 1 ? 's' : ''}</span>}
            {exclusions.length > 0 && <span className="flex items-center gap-1"><FileText size={12} className="text-red-400" />{exclusions.length} Exclusion{exclusions.length > 1 ? 's' : ''}</span>}
          </span>
          {expanded ? <ChevronUp size={15} /> : <ChevronDown size={15} />}
        </button>
      )}

      {/* Expanded detail */}
      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.2 }}
            className="overflow-hidden border-t border-slate-100">

            {/* Add-ons */}
            {addOns.length > 0 && (
              <div className="px-5 py-4 border-b border-slate-50">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3 flex items-center gap-1.5">
                  <Tag size={12} className="text-blue-500" /> Optional Add-Ons
                </p>
                <div>
                  {addOns.map((addon, i) => <AddOnRow key={i} addon={addon} />)}
                </div>
              </div>
            )}

            {/* NCD Discounts */}
            {discounts.length > 0 && (
              <div className="px-5 py-4 border-b border-slate-50">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3 flex items-center gap-1.5">
                  <Percent size={12} className="text-emerald-500" /> No-Claims Discount (NCD)
                </p>
                <NcdLadder discounts={discounts} />
                <p className="text-xs text-slate-400 mt-2">Earn higher discounts for each consecutive claim-free year.</p>
              </div>
            )}

            {/* Standard exclusions */}
            {exclusions.length > 0 && (
              <div className="px-5 py-4">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <FileText size={12} className="text-red-400" /> Standard Exclusions
                </p>
                <div className="flex flex-wrap gap-1.5">
                  {exclusions.map(e => <ExclusionChip key={e} code={e} />)}
                </div>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}

// ─── Offers tab — rich cards ──────────────────────────────────────────────────

const POLICY_IRI_LABELS = {
  claimPolicy: 'Claim Policy',
  cancellationPolicy: 'Cancellation Policy',
  renewalPolicy: 'Renewal Policy',
  endorsementPolicy: 'Endorsement Policy',
  grievanceSlaPolicy: 'Grievance SLA',
}

function OfferCard({ offer }) {
  const [expanded, setExpanded] = useState(false)
  const oa = offer.offer_attributes || {}
  const inclusions = oa.inclusions || []
  const requiredDocs = oa.requiredDocuments || []
  const specialExclusions = oa.specialExclusions || []
  const policyIris = Object.entries(POLICY_IRI_LABELS).filter(([k]) => oa[k])

  return (
    <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
      className="bg-white border border-slate-200 rounded-2xl overflow-hidden shadow-sm hover:shadow-md transition-shadow">

      {/* Header */}
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-mono text-slate-400 truncate">{offer.offer_attributes?.offerId || `#${offer.id}`}</p>
            {offer.resource_name && (
              <p className="text-base font-semibold text-slate-900 mt-0.5 leading-tight">{offer.resource_name}</p>
            )}
          </div>
          <ZoneBadge zone={offer.tariff_zone} />
        </div>

        {/* Rate row */}
        <div className="flex items-center gap-3 mt-3 flex-wrap">
          {offer.premium_rate_min != null && (
            <div className="flex items-center gap-1 text-sm">
              <Percent size={14} className="text-slate-400" />
              <span className="font-semibold text-slate-800">{offer.premium_rate_min}% – {offer.premium_rate_max}%</span>
              <span className="text-slate-400 text-xs">premium rate</span>
            </div>
          )}
          {requiredDocs.length > 0 && (
            <span className="text-xs px-2.5 py-1 rounded-lg bg-amber-50 border border-amber-200 text-amber-700 font-medium">
              {requiredDocs.length} required doc{requiredDocs.length > 1 ? 's' : ''}
            </span>
          )}
          {offer.valid_until && (
            <span className="text-xs text-slate-400">Valid until: {fmtDate(offer.valid_until)}</span>
          )}
        </div>

        {/* Linked resource badge */}
        {offer.resource_id && (
          <div className="mt-3 flex items-center gap-1.5 text-xs text-blue-600">
            <Link size={12} />
            <span>Resource ID: <span className="font-mono">{offer.resource_id}</span></span>
          </div>
        )}
      </div>

      {/* Inclusions (always visible) */}
      {inclusions.length > 0 && (
        <div className="px-5 py-3 border-t border-slate-50 bg-slate-50/50">
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
            <Shield size={12} /> Offer Inclusions
          </p>
          <div className="flex flex-wrap gap-1.5">
            {inclusions.map(c => <InclusionChip key={c} code={c} />)}
          </div>
        </div>
      )}

      {/* Expand toggle */}
      {(requiredDocs.length > 0 || specialExclusions.length > 0 || policyIris.length > 0) && (
        <button type="button" onClick={() => setExpanded(v => !v)}
          className="w-full flex items-center justify-between px-5 py-2.5 border-t border-slate-100 text-xs font-semibold text-slate-500 hover:bg-slate-50 transition-colors">
          <span className="flex items-center gap-3">
            {requiredDocs.length > 0 && <span className="flex items-center gap-1"><FileText size={12} className="text-amber-500" />{requiredDocs.length} Required Docs</span>}
            {specialExclusions.length > 0 && <span className="flex items-center gap-1"><FileText size={12} className="text-red-400" />{specialExclusions.length} Special Exclusions</span>}
            {policyIris.length > 0 && <span className="flex items-center gap-1"><Tag size={12} className="text-violet-500" />{policyIris.length} Policies</span>}
          </span>
          {expanded ? <ChevronUp size={15} /> : <ChevronDown size={15} />}
        </button>
      )}

      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.2 }}
            className="overflow-hidden border-t border-slate-100">

            {requiredDocs.length > 0 && (
              <div className="px-5 py-4 border-b border-slate-50">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <FileText size={12} className="text-amber-500" /> Required Documents
                </p>
                <ul className="space-y-1">
                  {requiredDocs.map((doc, i) => (
                    <li key={i} className="flex items-center gap-2 text-sm text-slate-700">
                      <span className="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" />{doc}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {specialExclusions.length > 0 && (
              <div className="px-5 py-4 border-b border-slate-50">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <FileText size={12} className="text-red-400" /> Special Exclusions
                </p>
                <ul className="space-y-1">
                  {specialExclusions.map((ex, i) => (
                    <li key={i} className="flex items-center gap-2 text-sm text-slate-600">
                      <span className="w-1.5 h-1.5 rounded-full bg-red-300 shrink-0" />{ex}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {policyIris.length > 0 && (
              <div className="px-5 py-4">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                  <Tag size={12} className="text-violet-500" /> Policy IRIs
                </p>
                <div className="space-y-1.5">
                  {policyIris.map(([key, label]) => (
                    <div key={key} className="flex items-center gap-2">
                      <span className="text-xs text-slate-400 w-32 shrink-0">{label}</span>
                      <span className="text-xs font-mono text-violet-700 truncate">{oa[key]}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}

// ─── Catalogs tab — rich cards ────────────────────────────────────────────────

const STATUS_STYLE = {
  ACCEPTED:    { cls: 'bg-emerald-100 text-emerald-800 border-emerald-200', dot: 'bg-emerald-500' },
  PENDING:     { cls: 'bg-amber-100 text-amber-800 border-amber-200',    dot: 'bg-amber-400' },
  REJECTED:    { cls: 'bg-red-100 text-red-800 border-red-200',          dot: 'bg-red-500' },
  UNPUBLISHED: { cls: 'bg-slate-100 text-slate-600 border-slate-200',    dot: 'bg-slate-400' },
}

function CatalogStatusBadge({ status }) {
  const s = STATUS_STYLE[status] || STATUS_STYLE.UNPUBLISHED
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs font-semibold ${s.cls}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${s.dot}`} />
      {status}
    </span>
  )
}

function CatalogCard({ catalog }) {
  return (
    <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
      className="bg-white border border-slate-200 rounded-2xl overflow-hidden shadow-sm hover:shadow-md transition-shadow">

      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-base font-semibold text-slate-900 leading-tight truncate">{catalog.name}</h3>
            <p className="text-xs font-mono text-slate-400 mt-0.5">ID #{catalog.id}</p>
          </div>
          <CatalogStatusBadge status={catalog.cds_status || 'UNPUBLISHED'} />
        </div>

        <div className="flex items-center gap-2 mt-3 flex-wrap">
          <span className="text-xs px-2.5 py-1 rounded-lg bg-blue-50 border border-blue-200 text-blue-700 font-semibold">v{catalog.version}</span>
          {catalog.provider_name && (
            <span className="flex items-center gap-1.5 text-xs text-slate-600 px-2.5 py-1 rounded-lg bg-slate-50 border border-slate-200">
              <Building2 size={11} className="text-slate-400" />
              {catalog.provider_name}
            </span>
          )}
          {catalog.created_at && (
            <span className="text-xs text-slate-400">
              Created {fmtDateTime(catalog.created_at)}
            </span>
          )}
        </div>
      </div>

      {catalog.provider_name && (
        <div className="px-5 py-3 border-t border-slate-50 bg-slate-50/40">
          <div className="flex items-center gap-2 text-xs text-slate-500">
            <Link size={12} className="text-blue-400" />
            <span>Linked provider: <span className="font-semibold text-slate-700">{catalog.provider_name}</span></span>
          </div>
        </div>
      )}
    </motion.div>
  )
}

// ─── By-Catalog linked view ───────────────────────────────────────────────────

function LinkedResourceRow({ product, offers }) {
  const [open, setOpen] = useState(false)
  const ra = product.resource_attributes || {}
  const name = ra.descriptor?.name || product.product_type || 'Unnamed Resource'
  const productType = product.product_type || ra.productType || ''
  const vehicleType = product.vehicle_type || ra.vehicleType || ''
  const ojkCode = product.ojk_product_code || ra.ojkProductCode

  return (
    <div className="border border-slate-100 rounded-xl overflow-hidden">
      <button type="button" onClick={() => setOpen(v => !v)}
        className="w-full flex items-center gap-3 px-4 py-3 bg-slate-50 hover:bg-slate-100 transition-colors text-left">
        <div className="p-1.5 bg-blue-100 rounded-lg shrink-0">
          <Box size={14} className="text-blue-600" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-slate-800 truncate">{name}</p>
          <p className="text-xs text-slate-400 font-mono">ID: {product.id}</p>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          {productType && <span className="text-xs bg-emerald-100 text-emerald-700 px-2 py-0.5 rounded-lg font-medium">{productType.replace('MOTOR_','')}</span>}
          {vehicleType && <span className="text-xs bg-blue-100 text-blue-700 px-2 py-0.5 rounded-lg font-medium">{vehicleType.replace('_WHEELER',' W').replace('COMMERCIAL_VEHICLE','Comm.')}</span>}
          <span className="text-xs bg-slate-200 text-slate-600 px-2 py-0.5 rounded-lg">{offers.length} offer{offers.length !== 1 ? 's' : ''}</span>
          {open ? <ChevronUp size={14} className="text-slate-400" /> : <ChevronDown size={14} className="text-slate-400" />}
        </div>
      </button>

      <AnimatePresence initial={false}>
        {open && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.15 }} className="overflow-hidden">
            {ojkCode && (
              <div className="px-4 py-2 border-t border-slate-100 bg-white flex items-center gap-2 text-xs text-slate-500">
                <Shield size={11} className="text-slate-400" />
                OJK Code: <span className="font-mono text-slate-700">{ojkCode}</span>
              </div>
            )}
            {offers.length > 0 ? (
              <div className="divide-y divide-slate-50 border-t border-slate-100">
                {offers.map(o => {
                  const oa = o.offer_attributes || {}
                  return (
                    <div key={o.id} className="px-4 py-2.5 bg-white flex items-center gap-3">
                      <Tag size={12} className="text-indigo-400 shrink-0" />
                      <div className="flex-1 min-w-0">
                        <span className="text-sm text-slate-700 font-medium">{o.tariff_zone || oa.tariffZone || `Offer #${o.id}`}</span>
                        {(o.premium_rate_min || o.premium_rate_max) && (
                          <span className="ml-2 text-xs text-slate-500">{o.premium_rate_min}%–{o.premium_rate_max}% p.a.</span>
                        )}
                      </div>
                      <span className="text-xs font-mono text-slate-400">ID: {o.id}</span>
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="px-4 py-2.5 text-xs text-slate-400 bg-white border-t border-slate-100">No offers linked to this product yet.</p>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function CatalogTreeCard({ catalog, products, offers, defaultOpen = false }) {
  const [open, setOpen] = useState(defaultOpen)
  const offersByResource = {}
  offers.forEach(o => {
    if (!offersByResource[o.resource_id]) offersByResource[o.resource_id] = []
    offersByResource[o.resource_id].push(o)
  })

  return (
    <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}
      className="bg-white border border-slate-200 rounded-2xl overflow-hidden shadow-sm">

      {/* Clickable catalog header */}
      <button type="button" onClick={() => setOpen(v => !v)}
        className="w-full px-5 py-4 bg-gradient-to-r from-slate-50 to-white flex items-start justify-between gap-3 hover:bg-slate-50 transition-colors text-left">
        <div className="flex items-center gap-3 min-w-0">
          <div className="p-2 bg-indigo-100 rounded-xl shrink-0">
            <Layers size={16} className="text-indigo-600" />
          </div>
          <div className="min-w-0">
            <h3 className="text-sm font-bold text-slate-900 truncate">{catalog.name}</h3>
            <div className="flex items-center gap-2 mt-0.5 flex-wrap">
              <span className="text-xs font-mono text-slate-400">ID #{catalog.id}</span>
              {catalog.provider_name && (
                <span className="flex items-center gap-1 text-xs text-slate-500">
                  <Building2 size={10} className="text-slate-400" />
                  {catalog.provider_name}
                </span>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <CatalogStatusBadge status={catalog.cds_status || 'UNPUBLISHED'} />
          <span className="text-xs text-slate-400 bg-slate-100 px-2 py-0.5 rounded-lg">{products.length} products · {offers.length} offers</span>
          {open ? <ChevronUp size={15} className="text-slate-400" /> : <ChevronDown size={15} className="text-slate-400" />}
        </div>
      </button>

      {/* Collapsible body */}
      <AnimatePresence initial={false}>
        {open && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.18 }} className="overflow-hidden">

            <div className="p-4 space-y-2 border-t border-slate-100">
              {products.length === 0 ? (
                <p className="text-xs text-slate-400 py-4 text-center">No products in this catalog yet.</p>
              ) : (
                products.map(p => (
                  <LinkedResourceRow key={p.id} product={p} offers={offersByResource[p.id] || []} />
                ))
              )}
            </div>

            <div className="px-5 py-3 border-t border-slate-100 bg-slate-50/50 flex items-center gap-4 text-xs text-slate-500">
              <span className="flex items-center gap-1.5"><Box size={11} className="text-blue-400" />{products.length} product{products.length !== 1 ? 's' : ''}</span>
              <span className="flex items-center gap-1.5"><Tag size={11} className="text-indigo-400" />{offers.length} offer{offers.length !== 1 ? 's' : ''}</span>
              {catalog.created_at && <span className="ml-auto">Created {fmtDateTime(catalog.created_at)}</span>}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}

function LinkedView({ catalogs, products, offers }) {
  if (catalogs.length === 0) {
    return <div className="py-16 text-center text-slate-400 text-sm bg-white rounded-2xl border border-slate-100">No catalogs yet. Publish a catalog first.</div>
  }

  return (
    <div className="space-y-3">
      {catalogs.map((catalog, ci) => (
        <CatalogTreeCard
          key={catalog.id || ci}
          catalog={catalog}
          products={products}
          offers={offers}
          defaultOpen={ci === 0}
        />
      ))}
    </div>
  )
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function InventoryPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState('products')
  const [products, setProducts] = useState([])
  const [offers, setOffers] = useState([])
  const [catalogs, setCatalogs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([
      client.get('/v1/inventory/resources'),
      client.get('/v1/inventory/offers'),
      client.get('/v1/catalogs'),
    ])
      .then(([r1, r2, r3]) => {
        setProducts(Array.isArray(r1.data) ? r1.data : (r1.data?.resources || r1.data?.items || []))
        setOffers(Array.isArray(r2.data) ? r2.data : (r2.data?.offers || r2.data?.items || []))
        setCatalogs(Array.isArray(r3.data) ? r3.data : (r3.data?.catalogs || r3.data?.items || []))
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-64 text-slate-500 text-sm">{t('common.loading')}</div>
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-900 mb-6">{t('inventory.title')}</h1>

      {error && <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">{error}</div>}

      <div className="flex gap-2 mb-5 flex-wrap">
        <Tab active={tab === 'products'} onClick={() => setTab('products')} label={t('inventory.products')} count={products.length} />
        <Tab active={tab === 'offers'} onClick={() => setTab('offers')} label={t('inventory.offers')} count={offers.length} />
        <Tab active={tab === 'catalogs'} onClick={() => setTab('catalogs')} label={t('inventory.catalogs')} count={catalogs.length} />
        <Tab active={tab === 'linked'} onClick={() => setTab('linked')} label="By Catalog" count={catalogs.length} />
      </div>

      {/* Products — card grid */}
      {tab === 'products' && (
        products.length === 0
          ? <div className="py-16 text-center text-slate-400 text-sm bg-white rounded-2xl border border-slate-100">{t('common.no_data')}</div>
          : (
            <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
              {products.map((p, i) => <ProductCard key={p.id || i} product={p} />)}
            </div>
          )
      )}

      {/* Offers — card grid */}
      {tab === 'offers' && (
        offers.length === 0
          ? <div className="py-16 text-center text-slate-400 text-sm bg-white rounded-2xl border border-slate-100">{t('common.no_data')}</div>
          : (
            <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
              {offers.map((o, i) => <OfferCard key={o.id || i} offer={o} />)}
            </div>
          )
      )}

      {/* Catalogs — card grid */}
      {tab === 'catalogs' && (
        catalogs.length === 0
          ? <div className="py-16 text-center text-slate-400 text-sm bg-white rounded-2xl border border-slate-100">{t('common.no_data')}</div>
          : (
            <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
              {catalogs.map((c, i) => <CatalogCard key={c.id || i} catalog={c} />)}
            </div>
          )
      )}

      {/* By Catalog — linked tree view */}
      {tab === 'linked' && (
        <LinkedView catalogs={catalogs} products={products} offers={offers} />
      )}
    </div>
  )
}
