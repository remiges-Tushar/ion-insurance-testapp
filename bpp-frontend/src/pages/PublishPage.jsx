import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { CheckCircle, ChevronRight, Wand2, AlertCircle, Plus, Trash2, X, Link2, Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import client from '../api/client.js'

function uniqueSuffix() {
  return Date.now().toString(36).toUpperCase().slice(-5)
}

// ─── Constants ───────────────────────────────────────────────────────────────

const OJK_RATES = {
  ZONE_1: { min: 3.82, max: 4.20 },
  ZONE_2: { min: 3.26, max: 3.59 },
  ZONE_3: { min: 2.53, max: 3.08 },
  ZONE_4: { min: 2.08, max: 2.29 },
  ZONE_5: { min: 1.54, max: 1.69 },
}

const COVERAGE_INCLUSIONS = [
  'PARTIAL_LOSS', 'TOTAL_LOSS', 'THEFT', 'FIRE', 'THIRD_PARTY_LIABILITY',
  'NATURAL_DISASTER', 'ROADSIDE_ASSISTANCE', 'PERSONAL_ACCIDENT_DRIVER',
  'PERSONAL_ACCIDENT_PASSENGER',
]

const STANDARD_EXCLUSIONS = [
  'OVERLOADING', 'MECHANICAL_BREAKDOWN', 'DUI', 'INTENTIONAL_DAMAGE',
  'TIRE_ONLY', 'RACING', 'WAR_TERRORISM', 'NUCLEAR',
]

// ─── Demo fill data (bilingual) ───────────────────────────────────────────────

const DEMO_DATA = {
  en: {
    provider: {
      id: 'PROV-ASURANSI-MAJU-001',
      descriptor: {
        name: 'PT Asuransi Maju Tbk',
        shortDesc: 'OJK-licensed general insurance company. Provides motor vehicle, property, and fire insurance products.',
        longDesc: 'PT Asuransi Maju Tbk is a general insurance company operating since 1985 with an OJK (Financial Services Authority) business license. Core products include comprehensive and TLO/TPL motor vehicle insurance compliant with OJK SE-06/D.05/2013.',
      },
      providerAttributes: {
        ojkLicenseNumber: 'KEP-112/KM.10/1985',
        ojkLicenseType: 'ASURANSI_UMUM',
        companyRegistrationNumber: 'AHU-0032451.AH.01.01.2005',
        nibNumber: '1234567890123',
        ojkIknbLicense: 'S-1045/NB.11/2020',
        claimsRatioPercent: 62.4,
        solvencyMarginPercent: 125.8,
      },
      availableAt: [{
        id: 'LOC-JAKARTA-HO',
        descriptor: { name: 'Jakarta Head Office' },
        address: { door: 'Floor 12', building: 'Menara Sudirman Tower', street: 'Jl. Jend. Sudirman Kav. 60', city: 'South Jakarta', state: 'DKI Jakarta', country: 'IDN', areaCode: '12190' },
        gps: '-6.2241,106.8074',
      }],
    },
    catalog: {
      id: 'CAT-ASURANSI-MAJU-2026',
      descriptor: { name: 'Motor Vehicle Insurance Catalog 2026' },
      version: '1.0.0',
    },
    resource: {
      id: 'RES-MOTOR-COMP-Z3-4W',
      descriptor: {
        name: 'Comprehensive Car Insurance — Zone 3',
        shortDesc: 'Full protection for four-wheeled vehicles. Covers accidents, theft, fire, and third-party liability.',
        longDesc: 'Comprehensive (All Risk) motor vehicle insurance for private four-wheeled vehicles with premiums based on OJK Zone 3 tariffs (DKI Jakarta, Banten, West Java). Insured Declared Value (IDV) is set based on market value minus depreciation per OJK SE-06/D.05/2013.',
      },
      category: { id: 'motor-four-wheeler', descriptor: { name: 'Four-Wheeled Vehicle' } },
      resourceAttributes: {
        productType: 'MOTOR_COMPREHENSIVE',
        vehicleType: 'FOUR_WHEELER',
        tariffZone: 'ZONE_3',
        premiumRateRange: { min: 1.05, max: 1.20 },
        ojkProductCode: 'AMJ-COMP-4W-Z3-2026',
        ojkLicenseNumber: 'KEP-112/KM.10/1985',
        coverageInclusions: ['PARTIAL_LOSS', 'TOTAL_LOSS', 'THEFT', 'FIRE', 'THIRD_PARTY_LIABILITY', 'NATURAL_DISASTER', 'ROADSIDE_ASSISTANCE'],
        standardExclusions: ['OVERLOADING', 'MECHANICAL_BREAKDOWN', 'DUI', 'INTENTIONAL_DAMAGE', 'TIRE_ONLY', 'RACING'],
        deductibleAmount: 300000,
        addOnOptions: [
          { code: 'ADD-ZERO-DEP', name: 'Zero Depreciation Cover', additionalPremiumRate: 0.15 },
          { code: 'ADD-RSA', name: '24-Hour Roadside Assistance', additionalPremiumRate: 0.05 },
          { code: 'ADD-PA-DRIVER', name: 'Personal Accident — Driver', additionalPremiumRate: 0.08 },
        ],
        discounts: [
          { type: 'NCD_1_YEAR', discountPercent: 5.0 },
          { type: 'NCD_2_YEAR', discountPercent: 10.0 },
          { type: 'NCD_3_YEAR', discountPercent: 15.0 },
        ],
      },
    },
    offer: {
      id: 'OFFER-MOTOR-COMP-Z3-4W-001',
      resourceIds: [],
      offerAttributes: {
        inclusions: ['PARTIAL_LOSS', 'TOTAL_LOSS', 'THEFT', 'FIRE', 'THIRD_PARTY_LIABILITY', 'NATURAL_DISASTER', 'ROADSIDE_ASSISTANCE'],
        specialExclusions: ['Vehicles with structural modifications not registered on STNK'],
        requiredDocuments: ['National ID (KTP / e-KTP) of primary driver', 'Valid STNK (vehicle registration)', 'Valid Class A driving licence (SIM A)', 'Vehicle photos from 4 sides (front, rear, left, right)'],
        claimPolicy: 'ion://policy/insurance/claim.motor.fnol-24h',
        cancellationPolicy: 'ion://policy/insurance/cancellation.motor.prorata',
        renewalPolicy: 'ion://policy/insurance/renewal.motor.30d-notice',
        endorsementPolicy: 'ion://policy/insurance/endorsement.motor.standard',
        grievanceSlaPolicy: 'ion://policy/insurance/grievance-sla.ojk.pojk23-2023',
      },
    },
  },
  id: {
    provider: {
      id: 'PROV-ASURANSI-MAJU-001',
      descriptor: {
        name: 'PT Asuransi Maju Tbk',
        shortDesc: 'Perusahaan asuransi umum berizin OJK. Menyediakan asuransi kendaraan bermotor, properti, dan kebakaran.',
        longDesc: 'PT Asuransi Maju Tbk adalah perusahaan asuransi umum yang telah beroperasi sejak 1985 dan memiliki izin usaha dari OJK (Otoritas Jasa Keuangan). Produk unggulan mencakup asuransi kendaraan bermotor komprehensif dan TLO/TPL yang sesuai dengan regulasi OJK SE-06/D.05/2013.',
      },
      providerAttributes: {
        ojkLicenseNumber: 'KEP-112/KM.10/1985',
        ojkLicenseType: 'ASURANSI_UMUM',
        companyRegistrationNumber: 'AHU-0032451.AH.01.01.2005',
        nibNumber: '1234567890123',
        ojkIknbLicense: 'S-1045/NB.11/2020',
        claimsRatioPercent: 62.4,
        solvencyMarginPercent: 125.8,
      },
      availableAt: [{
        id: 'LOC-JAKARTA-HO',
        descriptor: { name: 'Kantor Pusat Jakarta' },
        address: { door: 'Lantai 12', building: 'Gedung Menara Sudirman', street: 'Jl. Jend. Sudirman Kav. 60', city: 'Jakarta Selatan', state: 'DKI Jakarta', country: 'IDN', areaCode: '12190' },
        gps: '-6.2241,106.8074',
      }],
    },
    catalog: {
      id: 'CAT-ASURANSI-MAJU-2026',
      descriptor: { name: 'Katalog Asuransi Kendaraan Bermotor 2026' },
      version: '1.0.0',
    },
    resource: {
      id: 'RES-MOTOR-COMP-Z3-4W',
      descriptor: {
        name: 'Asuransi Mobil Komprehensif — Zone 3',
        shortDesc: 'Perlindungan menyeluruh untuk kendaraan roda empat. Mencakup kecelakaan, pencurian, kebakaran, dan tanggung gugat pihak ketiga.',
        longDesc: 'Asuransi kendaraan bermotor komprehensif (All Risk) untuk kendaraan roda empat non-niaga dengan premi sesuai tarif OJK Zone 3 (DKI Jakarta, Banten, Jawa Barat). Nilai pertanggungan (IDV) ditetapkan berdasarkan nilai pasar kendaraan dikurangi penyusutan sesuai OJK SE-06/D.05/2013.',
      },
      category: { id: 'motor-four-wheeler', descriptor: { name: 'Kendaraan Roda Empat' } },
      resourceAttributes: {
        productType: 'MOTOR_COMPREHENSIVE',
        vehicleType: 'FOUR_WHEELER',
        tariffZone: 'ZONE_3',
        premiumRateRange: { min: 1.05, max: 1.20 },
        ojkProductCode: 'AMJ-COMP-4W-Z3-2026',
        ojkLicenseNumber: 'KEP-112/KM.10/1985',
        coverageInclusions: ['PARTIAL_LOSS', 'TOTAL_LOSS', 'THEFT', 'FIRE', 'THIRD_PARTY_LIABILITY', 'NATURAL_DISASTER', 'ROADSIDE_ASSISTANCE'],
        standardExclusions: ['OVERLOADING', 'MECHANICAL_BREAKDOWN', 'DUI', 'INTENTIONAL_DAMAGE', 'TIRE_ONLY', 'RACING'],
        deductibleAmount: 300000,
        addOnOptions: [
          { code: 'ADD-ZERO-DEP', name: 'Perlindungan Tanpa Penyusutan (Zero Depreciation)', additionalPremiumRate: 0.15 },
          { code: 'ADD-RSA', name: 'Bantuan Darurat 24 Jam (Roadside Assistance)', additionalPremiumRate: 0.05 },
          { code: 'ADD-PA-DRIVER', name: 'Kecelakaan Diri Pengemudi', additionalPremiumRate: 0.08 },
        ],
        discounts: [
          { type: 'NCD_1_YEAR', discountPercent: 5.0 },
          { type: 'NCD_2_YEAR', discountPercent: 10.0 },
          { type: 'NCD_3_YEAR', discountPercent: 15.0 },
        ],
      },
    },
    offer: {
      id: 'OFFER-MOTOR-COMP-Z3-4W-001',
      resourceIds: [],
      offerAttributes: {
        inclusions: ['PARTIAL_LOSS', 'TOTAL_LOSS', 'THEFT', 'FIRE', 'THIRD_PARTY_LIABILITY', 'NATURAL_DISASTER', 'ROADSIDE_ASSISTANCE'],
        specialExclusions: ['Kendaraan dengan modifikasi struktural tidak terdaftar di STNK'],
        requiredDocuments: ['KTP / e-KTP pengemudi utama', 'STNK yang masih berlaku', 'SIM A yang masih berlaku', 'Foto kendaraan 4 sisi (depan, belakang, kiri, kanan)'],
        claimPolicy: 'ion://policy/insurance/claim.motor.fnol-24h',
        cancellationPolicy: 'ion://policy/insurance/cancellation.motor.prorata',
        renewalPolicy: 'ion://policy/insurance/renewal.motor.30d-notice',
        endorsementPolicy: 'ion://policy/insurance/endorsement.motor.standard',
        grievanceSlaPolicy: 'ion://policy/insurance/grievance-sla.ojk.pojk23-2023',
      },
    },
  },
}

function getDemoData(lang) {
  return DEMO_DATA[lang] || DEMO_DATA.en
}

// ─── Blank factories ──────────────────────────────────────────────────────────

function blankProvider() {
  return {
    id: '', descriptor: { name: '', shortDesc: '', longDesc: '' },
    providerAttributes: { ojkLicenseNumber: '', ojkLicenseType: 'ASURANSI_UMUM', companyRegistrationNumber: '', nibNumber: '', ojkIknbLicense: '', claimsRatioPercent: '', solvencyMarginPercent: '' },
    availableAt: [],
  }
}

function blankLocation() {
  return { id: '', descriptor: { name: '' }, address: { door: '', building: '', street: '', city: '', state: '', country: 'IDN', areaCode: '' }, gps: '' }
}

function blankCatalog() {
  return { id: '', descriptor: { name: '' }, version: '1.0.0', providerId: '' }
}

function blankResource(ojkLicense = '') {
  return {
    id: '', descriptor: { name: '', shortDesc: '', longDesc: '' },
    category: { id: '', descriptor: { name: '' } },
    providerId: '',
    resourceAttributes: {
      productType: 'MOTOR_COMPREHENSIVE', vehicleType: 'FOUR_WHEELER',
      tariffZone: 'ZONE_3', premiumRateRange: { min: OJK_RATES.ZONE_3.min, max: OJK_RATES.ZONE_3.max },
      ojkProductCode: '', ojkLicenseNumber: ojkLicense,
      coverageInclusions: [], standardExclusions: [],
      deductibleAmount: '', addOnOptions: [], discounts: [],
    },
  }
}

function blankOffer() {
  return {
    id: '', resourceIds: [],
    offerAttributes: { inclusions: [], specialExclusions: [], requiredDocuments: [], claimPolicy: '', cancellationPolicy: '', renewalPolicy: '', endorsementPolicy: '', grievanceSlaPolicy: '' },
  }
}

// ─── Deep-set utility ─────────────────────────────────────────────────────────

function setNested(obj, path, value) {
  const keys = path.split('.')
  const result = Array.isArray(obj) ? [...obj] : { ...obj }
  let curr = result
  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]
    curr[key] = Array.isArray(curr[key]) ? [...curr[key]] : { ...curr[key] }
    curr = curr[key]
  }
  curr[keys[keys.length - 1]] = value
  return result
}

// ─── UI primitives ────────────────────────────────────────────────────────────

function Label({ children, required }) {
  return (
    <label className="block text-sm font-medium text-slate-700 mb-1">
      {children}{required && <span className="text-red-500 ml-0.5">*</span>}
    </label>
  )
}

function Input({ className = '', ...props }) {
  return <input {...props} className={`w-full border border-slate-300 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 transition ${className}`} />
}

function Textarea({ ...props }) {
  return <textarea {...props} rows={3} className="w-full border border-slate-300 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 transition resize-none" />
}

function SelectInput({ children, ...props }) {
  return <select {...props} className="w-full border border-slate-300 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 transition bg-white">{children}</select>
}

function CheckGroup({ options, selected, onChange }) {
  return (
    <div className="grid grid-cols-2 gap-2">
      {options.map(opt => {
        const checked = selected.includes(opt)
        return (
          <label key={opt} className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer text-xs transition-all ${checked ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-slate-200 text-slate-600 hover:bg-slate-50'}`}>
            <input type="checkbox" checked={checked} onChange={() => onChange(checked ? selected.filter(s => s !== opt) : [...selected, opt])} className="accent-blue-600" />
            <span className="truncate">{opt.replace(/_/g, ' ')}</span>
          </label>
        )
      })}
    </div>
  )
}

function SectionHead({ children }) {
  return <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 mt-5 pb-1 border-b border-slate-100">{children}</h3>
}

function AutoFillBtn({ onClick }) {
  return (
    <button type="button" onClick={onClick} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-violet-700 bg-violet-50 border border-violet-200 rounded-lg hover:bg-violet-100 transition">
      <Wand2 size={13} /> Auto Fill
    </button>
  )
}

function parsePublishError(raw) {
  if (!raw) return null
  // Try to extract the inner onix JSON error from "... returned 4xx: {...}"
  const jsonMatch = raw.match(/returned \d+:\s*(\{[\s\S]+)$/)
  if (jsonMatch) {
    try {
      const obj = JSON.parse(jsonMatch[1])
      const err = obj?.message?.error
      if (err) {
        return {
          path: err.paths || null,
          message: (err.message || '').split('\n')[0], // first line only
          full: err.message,
        }
      }
    } catch { /* fall through */ }
  }
  return { message: raw, full: raw, path: null }
}

function ErrorBox({ detail }) {
  const [expanded, setExpanded] = React.useState(false)
  if (!detail) return null
  const parsed = parsePublishError(detail)
  const isLong = parsed?.full && parsed.full.length > 200

  return (
    <div className="mt-4 bg-red-50 border border-red-200 rounded-xl px-4 py-3">
      <div className="flex items-start gap-2">
        <AlertCircle size={16} className="text-red-500 mt-0.5 shrink-0" />
        <div className="min-w-0 flex-1">
          {parsed?.path && (
            <p className="text-xs font-semibold text-red-500 mb-1 font-mono">{parsed.path}</p>
          )}
          <p className="text-sm text-red-700">{parsed?.message || detail}</p>
          {isLong && (
            <button type="button" onClick={() => setExpanded(v => !v)}
              className="text-xs text-red-500 underline mt-1 hover:text-red-700">
              {expanded ? 'Show less' : 'Show full error'}
            </button>
          )}
          {expanded && (
            <pre className="mt-2 text-xs text-red-600 bg-red-100 rounded-lg p-2 overflow-x-auto whitespace-pre-wrap break-all">
              {parsed.full}
            </pre>
          )}
        </div>
      </div>
    </div>
  )
}

function StepIndicator({ current, total, labels }) {
  return (
    <div className="flex items-center gap-0 mb-8">
      {labels.map((label, i) => {
        const step = i + 1
        const done = step < current
        const active = step === current
        return (
          <React.Fragment key={i}>
            <div className="flex flex-col items-center">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-all ${done ? 'bg-green-500 text-white' : active ? 'bg-blue-600 text-white' : 'bg-slate-200 text-slate-500'}`}>
                {done ? <CheckCircle size={16} /> : step}
              </div>
              <span className={`text-xs mt-1 font-medium whitespace-nowrap ${active ? 'text-blue-600' : done ? 'text-green-600' : 'text-slate-400'}`}>{label}</span>
            </div>
            {i < total - 1 && <div className={`flex-1 h-0.5 mx-1 mb-5 ${done ? 'bg-green-400' : 'bg-slate-200'}`} />}
          </React.Fragment>
        )
      })}
    </div>
  )
}

// ─── Dynamic list editors ─────────────────────────────────────────────────────

function StringListEditor({ label, items, onChange, placeholder }) {
  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <Label>{label}</Label>
        <button type="button" onClick={() => onChange([...items, ''])} className="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700 font-medium">
          <Plus size={12} /> Add
        </button>
      </div>
      <div className="space-y-2">
        {items.map((item, i) => (
          <div key={i} className="flex gap-2">
            <Input value={item} onChange={e => onChange(items.map((x, idx) => idx === i ? e.target.value : x))} placeholder={placeholder} />
            <button type="button" onClick={() => onChange(items.filter((_, idx) => idx !== i))} className="text-slate-400 hover:text-red-500 shrink-0"><X size={16} /></button>
          </div>
        ))}
        {items.length === 0 && <p className="text-xs text-slate-400 italic">None — click Add</p>}
      </div>
    </div>
  )
}

function AddOnEditor({ items, onChange }) {
  function update(i, field, v) {
    onChange(items.map((x, idx) => idx === i ? { ...x, [field]: v } : x))
  }
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <Label>Add-On Options</Label>
        <button type="button" onClick={() => onChange([...items, { code: '', name: '', additionalPremiumRate: '' }])}
          className="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700 font-medium">
          <Plus size={12} /> Add
        </button>
      </div>
      <div className="space-y-2">
        {items.map((item, i) => (
          <div key={i} className="bg-slate-50 border border-slate-200 rounded-xl p-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-slate-500">Add-On {i + 1}</span>
              <button type="button" onClick={() => onChange(items.filter((_, idx) => idx !== i))}
                className="text-slate-300 hover:text-red-500 transition"><X size={14} /></button>
            </div>
            <div className="space-y-2">
              <div>
                <span className="text-xs text-slate-400 mb-0.5 block">Code</span>
                <Input value={item.code} onChange={e => update(i, 'code', e.target.value)} placeholder="ADD-NATURAL-DISASTER" />
              </div>
              <div>
                <span className="text-xs text-slate-400 mb-0.5 block">Name</span>
                <Input value={item.name} onChange={e => update(i, 'name', e.target.value)} placeholder="Perluasan Bencana Alam" />
              </div>
              <div>
                <span className="text-xs text-slate-400 mb-0.5 block">Additional Premium Rate (%)</span>
                <Input type="number" step="0.01" value={item.additionalPremiumRate} onChange={e => update(i, 'additionalPremiumRate', e.target.value)} placeholder="0.10" />
              </div>
            </div>
          </div>
        ))}
        {items.length === 0 && <p className="text-xs text-slate-400 italic">None — click Add</p>}
      </div>
    </div>
  )
}

const NCD_PRESET_TYPES = [
  { value: 'NCD_1_YEAR',      label: '1 Year No-Claims',   short: '1 yr' },
  { value: 'NCD_2_YEAR',      label: '2 Years No-Claims',  short: '2 yr' },
  { value: 'NCD_3_YEAR',      label: '3 Years No-Claims',  short: '3 yr' },
  { value: 'NCD_4_YEAR',      label: '4 Years No-Claims',  short: '4 yr' },
  { value: 'NCD_5_YEAR_PLUS', label: '5+ Years No-Claims', short: '5+ yr' },
  { value: 'LOYALTY_1_YEAR',  label: '1 Year Loyalty',     short: '1 yr loyal' },
  { value: 'LOYALTY_2_YEAR',  label: '2 Years Loyalty',    short: '2 yr loyal' },
  { value: 'OTHER',           label: 'Other (custom)',      short: 'custom' },
]

function DiscountEditor({ items, onChange }) {
  function update(i, field, v) {
    onChange(items.map((x, idx) => idx === i ? { ...x, [field]: v } : x))
  }

  // For the dropdown: if the stored type matches a preset use it; otherwise show 'OTHER'
  function selectValue(type) {
    return NCD_PRESET_TYPES.some(t => t.value === type && t.value !== 'OTHER') ? type : (type ? 'OTHER' : '')
  }

  // Items with enough data to show in the progression bar
  const filled = items.filter(d => d.type && d.discountPercent !== '')
    .sort((a, b) => Number(a.discountPercent) - Number(b.discountPercent))

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <Label>Discounts (NCD / Loyalty)</Label>
        <button type="button" onClick={() => onChange([...items, { type: '', discountPercent: '' }])}
          className="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700 font-medium">
          <Plus size={12} /> Add Level
        </button>
      </div>

      <div className="space-y-2">
        {items.map((item, i) => (
          <div key={i} className="bg-emerald-50 border border-emerald-200 rounded-xl p-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-emerald-700">Discount Level {i + 1}</span>
              <button type="button" onClick={() => onChange(items.filter((_, idx) => idx !== i))}
                className="text-slate-300 hover:text-red-500 transition"><X size={14} /></button>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <span className="text-xs text-slate-500 mb-0.5 block">Discount Type</span>
                <SelectInput value={selectValue(item.type)} onChange={e => update(i, 'type', e.target.value === 'OTHER' ? '' : e.target.value)}>
                  <option value="">Select type…</option>
                  {NCD_PRESET_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                </SelectInput>
                {selectValue(item.type) === 'OTHER' && (
                  <Input className="mt-1.5" value={item.type} onChange={e => update(i, 'type', e.target.value)} placeholder="e.g. FLEET_DISCOUNT" />
                )}
              </div>
              <div>
                <span className="text-xs text-slate-500 mb-0.5 block">Discount (%)</span>
                <div className="relative">
                  <Input type="number" step="0.1" min="0" max="100" value={item.discountPercent}
                    onChange={e => update(i, 'discountPercent', e.target.value)} placeholder="5.0" />
                  {item.discountPercent !== '' && (
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-emerald-600 font-bold text-sm pointer-events-none">%</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        ))}
        {items.length === 0 && <p className="text-xs text-slate-400 italic">None — click Add Level</p>}
      </div>

      {/* Live progression preview — shows once 2+ levels have values */}
      {filled.length >= 2 && (
        <div className="mt-3 bg-white border border-emerald-200 rounded-xl px-4 py-3">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2">Discount Progression</p>
          <div className="flex items-end gap-1 flex-wrap">
            {filled.map((d, i) => {
              const preset = NCD_PRESET_TYPES.find(t => t.value === d.type)
              const shortLabel = preset ? preset.short : (d.type?.replace(/_/g, ' ').toLowerCase() || '—')
              return (
                <React.Fragment key={i}>
                  <div className="flex flex-col items-center">
                    <span className="text-base font-bold text-emerald-700 leading-none">{d.discountPercent}%</span>
                    <span className="text-xs text-slate-400 mt-0.5 whitespace-nowrap">{shortLabel}</span>
                  </div>
                  {i < filled.length - 1 && (
                    <span className="text-slate-300 pb-4 mx-0.5 text-sm">→</span>
                  )}
                </React.Fragment>
              )
            })}
          </div>
          <p className="text-xs text-slate-400 mt-2">Earned for each consecutive claim-free year.</p>
        </div>
      )}
    </div>
  )
}

// ─── Shared: item card + "add" panel pattern ──────────────────────────────────

function ItemCard({ title, subtitle, meta, onRemove, onEdit, badge }) {
  return (
    <div className="bg-slate-50 border border-slate-200 rounded-xl px-4 py-3 flex items-start justify-between gap-3">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <p className="text-sm font-semibold text-slate-800 truncate">{title}</p>
          {badge && <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-amber-100 text-amber-700 shrink-0">{badge}</span>}
        </div>
        {subtitle && <p className="text-xs text-slate-500 mt-0.5 truncate">{subtitle}</p>}
        {meta && <p className="text-xs text-slate-400 mt-0.5">{meta}</p>}
      </div>
      <div className="flex items-center gap-1 shrink-0 mt-0.5">
        {onEdit && <button type="button" onClick={onEdit} className="text-slate-300 hover:text-blue-500 transition" title="Edit"><Pencil size={14} /></button>}
        <button type="button" onClick={onRemove} className="text-slate-300 hover:text-red-500 transition" title="Remove"><Trash2 size={15} /></button>
      </div>
    </div>
  )
}

function FieldError({ msg }) {
  if (!msg) return null
  return (
    <div className="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-200 rounded-xl text-xs text-red-700">
      <AlertCircle size={13} className="shrink-0" /> {msg}
    </div>
  )
}

function AddPanel({ title, isEditing, children, onAdd, addLabel, onCancel, canAdd = true, fieldError }) {
  return (
    <div className="border border-blue-200 bg-blue-50/30 rounded-2xl p-5 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-700">{isEditing ? `Edit ${title}` : title}</h3>
        {onCancel && <button type="button" onClick={onCancel} className="text-slate-400 hover:text-slate-600"><X size={17} /></button>}
      </div>
      {children}
      <FieldError msg={fieldError} />
      <div className="flex justify-end gap-3 pt-1 border-t border-blue-100 mt-4">
        {onCancel && <button type="button" onClick={onCancel} className="px-4 py-2 text-sm text-slate-600 border border-slate-200 bg-white rounded-xl hover:bg-slate-50">Cancel</button>}
        <button type="button" onClick={onAdd} disabled={!canAdd}
          className="flex items-center gap-2 px-4 py-2 text-sm font-semibold bg-blue-600 text-white rounded-xl hover:bg-blue-700 disabled:opacity-40 transition">
          <Plus size={14} /> {isEditing ? `Save Changes` : addLabel}
        </button>
      </div>
    </div>
  )
}

function AddMoreButton({ label, onClick }) {
  return (
    <button type="button" onClick={onClick}
      className="flex items-center gap-2 px-4 py-2.5 border border-dashed border-blue-300 rounded-xl text-sm text-blue-600 hover:bg-blue-50 w-full justify-center transition">
      <Plus size={16} /> {label}
    </button>
  )
}

function StepShell({ title, count, countLabel, children }) {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
        {count > 0 && <span className="text-xs font-medium px-2.5 py-1 rounded-full bg-green-100 text-green-700">{count} {countLabel}</span>}
      </div>
      {children}
    </div>
  )
}

// ─── Step 1: Providers ────────────────────────────────────────────────────────

function ProviderFormFields({ draft, set }) {
  function addLocation() { set('availableAt', [...draft.availableAt, blankLocation()]) }
  function removeLocation(i) { set('availableAt', draft.availableAt.filter((_, idx) => idx !== i)) }
  function setLoc(i, path, v) { set('availableAt', draft.availableAt.map((loc, idx) => idx === i ? setNested(loc, path, v) : loc)) }

  return (
    <>
      <SectionHead>Basic Info</SectionHead>
      <div className="grid grid-cols-2 gap-3">
        <div className="col-span-2">
          <Label required>Provider ID</Label>
          <Input value={draft.id} onChange={e => set('id', e.target.value)} placeholder="PROV-COMPANY-001" />
        </div>
        <div className="col-span-2">
          <Label required>Company Name</Label>
          <Input value={draft.descriptor.name} onChange={e => set('descriptor.name', e.target.value)} placeholder="PT Asuransi Maju Tbk" />
        </div>
        <div className="col-span-2">
          <Label>Short Description</Label>
          <Input value={draft.descriptor.shortDesc} onChange={e => set('descriptor.shortDesc', e.target.value)} placeholder="Brief company description" />
        </div>
        <div className="col-span-2">
          <Label>Long Description</Label>
          <Textarea value={draft.descriptor.longDesc} onChange={e => set('descriptor.longDesc', e.target.value)} placeholder="Full company description" />
        </div>
      </div>

      <SectionHead>OJK Regulatory</SectionHead>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label required>OJK License Number</Label>
          <Input value={draft.providerAttributes.ojkLicenseNumber} onChange={e => set('providerAttributes.ojkLicenseNumber', e.target.value)} placeholder="KEP-112/KM.10/1985" />
        </div>
        <div>
          <Label>License Type</Label>
          <SelectInput value={draft.providerAttributes.ojkLicenseType} onChange={e => set('providerAttributes.ojkLicenseType', e.target.value)}>
            <option value="ASURANSI_UMUM">ASURANSI_UMUM</option>
            <option value="ASURANSI_JIWA">ASURANSI_JIWA</option>
            <option value="REASURANSI">REASURANSI</option>
          </SelectInput>
        </div>
        <div>
          <Label>AHU Reg. Number</Label>
          <Input value={draft.providerAttributes.companyRegistrationNumber} onChange={e => set('providerAttributes.companyRegistrationNumber', e.target.value)} placeholder="AHU-0032451..." />
        </div>
        <div>
          <Label>NIB Number</Label>
          <Input value={draft.providerAttributes.nibNumber} onChange={e => set('providerAttributes.nibNumber', e.target.value)} placeholder="1234567890123" />
        </div>
        <div>
          <Label>OJK IKNB License</Label>
          <Input value={draft.providerAttributes.ojkIknbLicense} onChange={e => set('providerAttributes.ojkIknbLicense', e.target.value)} placeholder="S-1045/NB.11/2020" />
        </div>
        <div className="grid grid-cols-2 gap-2 col-span-2">
          <div>
            <Label>Claims Ratio (%)</Label>
            <Input type="number" step="0.1" value={draft.providerAttributes.claimsRatioPercent} onChange={e => set('providerAttributes.claimsRatioPercent', e.target.value)} placeholder="62.4" />
          </div>
          <div>
            <Label>Solvency Margin (%)</Label>
            <Input type="number" step="0.1" value={draft.providerAttributes.solvencyMarginPercent} onChange={e => set('providerAttributes.solvencyMarginPercent', e.target.value)} placeholder="125.8" />
          </div>
        </div>
      </div>

      <SectionHead>Office Locations</SectionHead>
      {draft.availableAt.map((loc, i) => (
        <div key={i} className="bg-white rounded-xl p-3 border border-slate-200 space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Location {i + 1}</span>
            <button type="button" onClick={() => removeLocation(i)} className="text-slate-300 hover:text-red-500"><Trash2 size={13} /></button>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div><Label>Location ID</Label><Input value={loc.id} onChange={e => setLoc(i, 'id', e.target.value)} placeholder="LOC-JKT-HO" /></div>
            <div><Label>Name</Label><Input value={loc.descriptor.name} onChange={e => setLoc(i, 'descriptor.name', e.target.value)} placeholder="Kantor Pusat Jakarta" /></div>
            <div><Label>Door / Unit</Label><Input value={loc.address.door} onChange={e => setLoc(i, 'address.door', e.target.value)} placeholder="Lantai 12" /></div>
            <div><Label>Building</Label><Input value={loc.address.building} onChange={e => setLoc(i, 'address.building', e.target.value)} placeholder="Gedung Menara Sudirman" /></div>
            <div className="col-span-2"><Label>Street</Label><Input value={loc.address.street} onChange={e => setLoc(i, 'address.street', e.target.value)} placeholder="Jl. Jend. Sudirman Kav. 60" /></div>
            <div><Label>City</Label><Input value={loc.address.city} onChange={e => setLoc(i, 'address.city', e.target.value)} placeholder="Jakarta Selatan" /></div>
            <div><Label>Province</Label><Input value={loc.address.state} onChange={e => setLoc(i, 'address.state', e.target.value)} placeholder="DKI Jakarta" /></div>
            <div><Label>Area Code</Label><Input value={loc.address.areaCode} onChange={e => setLoc(i, 'address.areaCode', e.target.value)} placeholder="12190" /></div>
            <div><Label>GPS</Label><Input value={loc.gps} onChange={e => setLoc(i, 'gps', e.target.value)} placeholder="-6.2241,106.8074" /></div>
          </div>
        </div>
      ))}
      <button type="button" onClick={addLocation}
        className="flex items-center gap-2 px-4 py-2 border border-dashed border-slate-300 rounded-xl text-xs text-slate-500 hover:bg-slate-50 w-full justify-center transition">
        <Plus size={14} /> Add Location
      </button>
    </>
  )
}

function ProvidersStep({ providers, setProviders, lang }) {
  const [drafting, setDrafting] = useState(false)
  const [editIdx, setEditIdx] = useState(null)
  const [draft, setDraft] = useState(blankProvider())
  const [fieldError, setFieldError] = useState('')

  const set = (path, v) => setDraft(d => typeof v !== 'undefined' && path.includes('.') ? setNested(d, path, v) : { ...d, [path]: v })

  function startEdit(i) {
    setEditIdx(i); setDraft({ ...providers[i] }); setDrafting(true); setFieldError('')
  }

  function handleCancel() {
    setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function handleAdd() {
    const id = draft.id.trim()
    if (!id || !draft.descriptor.name.trim()) return
    if (providers.some((p, i) => p.id === id && i !== editIdx)) {
      setFieldError(`ID "${id}" is already used by another provider.`); return
    }
    if (editIdx !== null) {
      setProviders(p => p.map((x, i) => i === editIdx ? draft : x))
    } else {
      setProviders(p => [...p, draft])
    }
    setDraft(blankProvider()); setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function autoFill() {
    const sfx = uniqueSuffix()
    const demo = getDemoData(lang).provider
    setDraft({
      ...demo,
      id: `${demo.id}-${sfx}`,
      availableAt: (demo.availableAt || []).map(loc => ({ ...loc, id: `${loc.id}-${sfx}` })),
    })
  }

  return (
    <StepShell title="Providers" count={providers.length} countLabel="added">
      {providers.map((prov, i) => (
        <ItemCard key={i}
          title={prov.descriptor.name}
          subtitle={`ID: ${prov.id}`}
          meta={prov.providerAttributes.ojkLicenseNumber ? `OJK: ${prov.providerAttributes.ojkLicenseNumber}` : undefined}
          onEdit={() => startEdit(i)}
          onRemove={() => setProviders(p => p.filter((_, idx) => idx !== i))}
        />
      ))}

      {drafting ? (
        <AddPanel title="Provider" isEditing={editIdx !== null} addLabel="Add Provider" onAdd={handleAdd} onCancel={handleCancel}
          canAdd={draft.id.trim().length > 0 && draft.descriptor.name.trim().length > 0} fieldError={fieldError}>
          <div className="flex justify-end -mt-2 -mb-2">
            <AutoFillBtn onClick={autoFill} />
          </div>
          <ProviderFormFields draft={draft} set={set} />
        </AddPanel>
      ) : (
        <AddMoreButton label="Add Provider" onClick={() => { setDraft(blankProvider()); setDrafting(true) }} />
      )}

      {providers.length === 0 && !drafting && (
        <p className="text-center text-xs text-slate-400 py-4">Create at least one provider. Catalogs will be linked to a provider.</p>
      )}
    </StepShell>
  )
}

// ─── Step 2: Catalogs (linked to a provider) ──────────────────────────────────

function CatalogsStep({ catalogs, setCatalogs, providers, lang }) {
  const [drafting, setDrafting] = useState(false)
  const [editIdx, setEditIdx] = useState(null)
  const [draft, setDraft] = useState(blankCatalog())
  const [fieldError, setFieldError] = useState('')

  function startEdit(i) {
    setEditIdx(i); setDraft({ ...catalogs[i] }); setDrafting(true); setFieldError('')
  }

  function handleCancel() {
    setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function handleAdd() {
    const id = draft.id.trim()
    if (!id || !draft.descriptor.name.trim() || !draft.providerId) return
    if (catalogs.some((c, i) => c.id === id && i !== editIdx)) {
      setFieldError(`ID "${id}" is already used by another catalog.`); return
    }
    if (editIdx !== null) {
      setCatalogs(c => c.map((x, i) => i === editIdx ? draft : x))
    } else {
      setCatalogs(c => [...c, draft])
    }
    setDraft(blankCatalog()); setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function autoFill() {
    const sfx = uniqueSuffix()
    const demo = getDemoData(lang).catalog
    setDraft({ ...demo, id: `${demo.id}-${sfx}`, providerId: providers[0]?.id || '' })
  }

  const linkedProvider = (id) => providers.find(p => p.id === id)

  return (
    <StepShell title="Catalogs" count={catalogs.length} countLabel="added">
      {catalogs.map((cat, i) => {
        const prov = linkedProvider(cat.providerId)
        return (
          <ItemCard key={i}
            title={cat.descriptor.name}
            subtitle={`ID: ${cat.id} · v${cat.version}`}
            meta={prov ? `Provider: ${prov.descriptor.name}` : `Provider: ${cat.providerId}`}
            badge={prov ? undefined : 'Unlinked'}
            onEdit={() => startEdit(i)}
            onRemove={() => setCatalogs(c => c.filter((_, idx) => idx !== i))}
          />
        )
      })}

      {drafting ? (
        <AddPanel title="Catalog" isEditing={editIdx !== null} addLabel="Add Catalog" onAdd={handleAdd} onCancel={handleCancel}
          canAdd={draft.id.trim().length > 0 && draft.descriptor.name.trim().length > 0 && !!draft.providerId} fieldError={fieldError}>
          <div className="flex justify-end -mt-2 -mb-2">
            <AutoFillBtn onClick={autoFill} />
          </div>
          <div className="space-y-3">
            <div>
              <Label required>Catalog ID</Label>
              <Input value={draft.id} onChange={e => setDraft(d => ({ ...d, id: e.target.value }))} placeholder="CAT-COMPANY-2026" />
            </div>
            <div>
              <Label required>Catalog Name</Label>
              <Input value={draft.descriptor.name} onChange={e => setDraft(d => setNested(d, 'descriptor.name', e.target.value))} placeholder="Motor Vehicle Insurance Catalog 2026" />
            </div>
            <div>
              <Label required>Version</Label>
              <Input value={draft.version} onChange={e => setDraft(d => ({ ...d, version: e.target.value }))} placeholder="1.0.0" />
            </div>
            <div>
              <Label required>Provider</Label>
              {providers.length === 0 ? (
                <div className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-amber-200 bg-amber-50 text-xs text-amber-700">
                  <AlertCircle size={14} /> No providers yet — go back to Step 1 to add one.
                </div>
              ) : (
                <SelectInput value={draft.providerId} onChange={e => setDraft(d => ({ ...d, providerId: e.target.value }))}>
                  <option value="">Select a provider…</option>
                  {providers.map(p => (
                    <option key={p.id} value={p.id}>{p.descriptor.name} ({p.id})</option>
                  ))}
                </SelectInput>
              )}
            </div>
            {draft.providerId && (
              <div className="flex items-center gap-2 px-3 py-2 bg-green-50 border border-green-200 rounded-xl text-xs text-green-700">
                <Link2 size={13} /> Linked to: {linkedProvider(draft.providerId)?.descriptor.name}
              </div>
            )}
          </div>
        </AddPanel>
      ) : (
        <AddMoreButton label="Add Catalog" onClick={() => { setDraft(blankCatalog()); setDrafting(true) }} />
      )}

      {catalogs.length === 0 && !drafting && (
        <p className="text-center text-xs text-slate-400 py-4">Each catalog is owned by one provider. You can add multiple catalogs.</p>
      )}
    </StepShell>
  )
}

// ─── Step 3: Resources ────────────────────────────────────────────────────────

function ResourceFormFields({ draft, set }) {
  const ra = draft.resourceAttributes

  function handleZone(zone) {
    set('resourceAttributes.tariffZone', zone)
    if (OJK_RATES[zone]) {
      set('resourceAttributes.premiumRateRange', { min: OJK_RATES[zone].min, max: OJK_RATES[zone].max })
    }
  }

  return (
    <>
      <SectionHead>Descriptor</SectionHead>
      <div className="space-y-3">
        <div><Label required>Resource ID</Label><Input value={draft.id} onChange={e => set('id', e.target.value)} placeholder="RES-MOTOR-COMP-Z3-4W" /></div>
        <div><Label required>Name</Label><Input value={draft.descriptor.name} onChange={e => set('descriptor.name', e.target.value)} placeholder="Asuransi Mobil Komprehensif — Zone 3" /></div>
        <div><Label>Short Description</Label><Input value={draft.descriptor.shortDesc} onChange={e => set('descriptor.shortDesc', e.target.value)} placeholder="Brief product description" /></div>
        <div><Label>Long Description</Label><Textarea value={draft.descriptor.longDesc} onChange={e => set('descriptor.longDesc', e.target.value)} placeholder="Detailed product description" /></div>
      </div>

      <SectionHead>Category</SectionHead>
      <div className="grid grid-cols-2 gap-3">
        <div><Label>Category ID</Label><Input value={draft.category.id} onChange={e => set('category.id', e.target.value)} placeholder="motor-four-wheeler" /></div>
        <div><Label>Category Name</Label><Input value={draft.category.descriptor.name} onChange={e => set('category.descriptor.name', e.target.value)} placeholder="Kendaraan Roda Empat" /></div>
      </div>

      <SectionHead>Coverage Profile</SectionHead>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label required>Product Type</Label>
          <SelectInput value={ra.productType} onChange={e => set('resourceAttributes.productType', e.target.value)}>
            <option value="MOTOR_COMPREHENSIVE">MOTOR_COMPREHENSIVE</option>
            <option value="MOTOR_THIRD_PARTY">MOTOR_THIRD_PARTY</option>
            <option value="MOTOR_FIRE_THEFT">MOTOR_FIRE_THEFT</option>
          </SelectInput>
        </div>
        <div>
          <Label required>Vehicle Type</Label>
          <SelectInput value={ra.vehicleType} onChange={e => set('resourceAttributes.vehicleType', e.target.value)}>
            <option value="TWO_WHEELER">TWO_WHEELER</option>
            <option value="FOUR_WHEELER">FOUR_WHEELER</option>
            <option value="COMMERCIAL_VEHICLE">COMMERCIAL_VEHICLE</option>
          </SelectInput>
        </div>
        <div>
          <Label required>Tariff Zone</Label>
          <SelectInput value={ra.tariffZone} onChange={e => handleZone(e.target.value)}>
            <option value="ZONE_1">ZONE_1 (3.82% – 4.20%)</option>
            <option value="ZONE_2">ZONE_2 (3.26% – 3.59%)</option>
            <option value="ZONE_3">ZONE_3 (2.53% – 3.08%)</option>
            <option value="ZONE_4">ZONE_4 (2.08% – 2.29%)</option>
            <option value="ZONE_5">ZONE_5 (1.54% – 1.69%)</option>
          </SelectInput>
        </div>
        <div><Label>Deductible (IDR)</Label><Input type="number" value={ra.deductibleAmount} onChange={e => set('resourceAttributes.deductibleAmount', e.target.value)} placeholder="300000" /></div>
        <div><Label required>Rate Min (%)</Label><Input type="number" step="0.01" value={ra.premiumRateRange.min} onChange={e => set('resourceAttributes.premiumRateRange.min', e.target.value)} /></div>
        <div><Label required>Rate Max (%)</Label><Input type="number" step="0.01" value={ra.premiumRateRange.max} onChange={e => set('resourceAttributes.premiumRateRange.max', e.target.value)} /></div>
      </div>

      <SectionHead>OJK Details</SectionHead>
      <div className="grid grid-cols-2 gap-3">
        <div><Label required>OJK Product Code</Label><Input value={ra.ojkProductCode} onChange={e => set('resourceAttributes.ojkProductCode', e.target.value)} placeholder="AMJ-COMP-4W-Z3-2026" /></div>
        <div><Label required>OJK License Number</Label><Input value={ra.ojkLicenseNumber} onChange={e => set('resourceAttributes.ojkLicenseNumber', e.target.value)} placeholder="KEP-112/KM.10/1985" /></div>
      </div>

      <SectionHead>Coverage Inclusions</SectionHead>
      <CheckGroup options={COVERAGE_INCLUSIONS} selected={ra.coverageInclusions} onChange={v => set('resourceAttributes.coverageInclusions', v)} />

      <SectionHead>Standard Exclusions</SectionHead>
      <CheckGroup options={STANDARD_EXCLUSIONS} selected={ra.standardExclusions} onChange={v => set('resourceAttributes.standardExclusions', v)} />

      <SectionHead>Add-Ons & Discounts</SectionHead>
      <AddOnEditor items={ra.addOnOptions} onChange={v => set('resourceAttributes.addOnOptions', v)} />
      <DiscountEditor items={ra.discounts} onChange={v => set('resourceAttributes.discounts', v)} />
    </>
  )
}

function ResourcesStep({ resources, setResources, providers, lang }) {
  const [drafting, setDrafting] = useState(false)
  const [editIdx, setEditIdx] = useState(null)
  const [draft, setDraft] = useState(blankResource())
  const [fieldError, setFieldError] = useState('')

  const set = (path, v) => setDraft(d => setNested(d, path, v))

  function startEdit(i) {
    setEditIdx(i); setDraft({ ...resources[i] }); setDrafting(true); setFieldError('')
  }

  function handleCancel() {
    setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function handleAdd() {
    const id = draft.id.trim()
    if (!id || !draft.descriptor.name.trim()) return
    if (resources.some((r, i) => r.id === id && i !== editIdx)) {
      setFieldError(`ID "${id}" is already used by another resource.`); return
    }
    if (editIdx !== null) {
      setResources(r => r.map((x, i) => i === editIdx ? draft : x))
    } else {
      setResources(r => [...r, draft])
    }
    setDraft(blankResource()); setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function autoFill() {
    const sfx = uniqueSuffix()
    const demo = getDemoData(lang).resource
    const ojkLicense = providers[0]?.providerAttributes?.ojkLicenseNumber || demo.resourceAttributes.ojkLicenseNumber
    const providerId = providers[0]?.id || ''
    setDraft({
      ...demo,
      id: `${demo.id}-${sfx}`,
      providerId,
      resourceAttributes: { ...demo.resourceAttributes, ojkLicenseNumber: ojkLicense },
    })
  }

  const linkedProvider = (id) => providers.find(p => p.id === id)

  return (
    <StepShell title="Resources" count={resources.length} countLabel="added">
      {resources.map((res, i) => {
        const prov = linkedProvider(res.providerId)
        return (
          <ItemCard key={i}
            title={res.descriptor.name || res.id}
            subtitle={`${res.resourceAttributes.productType} · ${res.resourceAttributes.vehicleType} · ${res.resourceAttributes.tariffZone}`}
            meta={`ID: ${res.id}${prov ? ` · Provider: ${prov.descriptor.name}` : ''}`}
            onEdit={() => startEdit(i)}
            onRemove={() => setResources(r => r.filter((_, idx) => idx !== i))}
          />
        )
      })}

      {drafting ? (
        <AddPanel title="Resource" isEditing={editIdx !== null} addLabel="Add Resource" onAdd={handleAdd} onCancel={handleCancel}
          canAdd={draft.id.trim().length > 0 && draft.descriptor.name.trim().length > 0} fieldError={fieldError}>
          <div className="flex justify-end -mt-2 -mb-2">
            <AutoFillBtn onClick={autoFill} />
          </div>
          {providers.length > 0 && (
            <div>
              <Label>Provider</Label>
              <SelectInput value={draft.providerId} onChange={e => {
                const prov = providers.find(p => p.id === e.target.value)
                setDraft(d => ({ ...d, providerId: e.target.value, resourceAttributes: { ...d.resourceAttributes, ojkLicenseNumber: prov?.providerAttributes?.ojkLicenseNumber || d.resourceAttributes.ojkLicenseNumber } }))
              }}>
                <option value="">Select provider (optional)…</option>
                {providers.map(p => <option key={p.id} value={p.id}>{p.descriptor.name}</option>)}
              </SelectInput>
            </div>
          )}
          <ResourceFormFields draft={draft} set={set} />
        </AddPanel>
      ) : (
        <AddMoreButton label="Add Resource" onClick={() => { setDraft(blankResource(providers[0]?.providerAttributes?.ojkLicenseNumber || '')); setDrafting(true) }} />
      )}
    </StepShell>
  )
}

// ─── Step 4: Offers ───────────────────────────────────────────────────────────

function OffersStep({ offers, setOffers, resources, lang }) {
  const [drafting, setDrafting] = useState(false)
  const [editIdx, setEditIdx] = useState(null)
  const [draft, setDraft] = useState(blankOffer())
  const [fieldError, setFieldError] = useState('')

  const set = (path, v) => setDraft(d => setNested(d, path, v))

  function startEdit(i) {
    setEditIdx(i); setDraft({ ...offers[i] }); setDrafting(true); setFieldError('')
  }

  function handleCancel() {
    setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function handleAdd() {
    const id = draft.id.trim()
    if (!id) return
    if (offers.some((o, i) => o.id === id && i !== editIdx)) {
      setFieldError(`ID "${id}" is already used by another offer.`); return
    }
    if (editIdx !== null) {
      setOffers(o => o.map((x, i) => i === editIdx ? draft : x))
    } else {
      setOffers(o => [...o, draft])
    }
    setDraft(blankOffer()); setDrafting(false); setEditIdx(null); setFieldError('')
  }

  function autoFill() {
    const sfx = uniqueSuffix()
    const demo = getDemoData(lang).offer
    setDraft({
      ...demo,
      id: `${demo.id}-${sfx}`,
      resourceIds: resources.length > 0 ? [resources[0].id] : [],
      offerAttributes: { ...demo.offerAttributes, inclusions: resources[0]?.resourceAttributes?.coverageInclusions?.slice(0, 5) || demo.offerAttributes.inclusions },
    })
  }

  function toggleResource(id) {
    const ids = draft.resourceIds.includes(id) ? draft.resourceIds.filter(x => x !== id) : [...draft.resourceIds, id]
    setDraft(d => ({ ...d, resourceIds: ids, offerAttributes: { ...d.offerAttributes, inclusions: [] } }))
  }

  const linkedInclusions = resources.filter(r => draft.resourceIds.includes(r.id)).flatMap(r => r.resourceAttributes.coverageInclusions)
  const availableInclusions = linkedInclusions.length > 0 ? [...new Set(linkedInclusions)] : COVERAGE_INCLUSIONS

  const oa = draft.offerAttributes

  return (
    <StepShell title="Offers" count={offers.length} countLabel="added">
      {offers.map((offer, i) => {
        const linkedRes = resources.filter(r => offer.resourceIds.includes(r.id))
        return (
          <ItemCard key={i}
            title={offer.id}
            subtitle={linkedRes.length > 0 ? `Resources: ${linkedRes.map(r => r.descriptor.name || r.id).join(', ')}` : 'No resources linked'}
            meta={`${offer.offerAttributes.inclusions.length} inclusions · ${offer.offerAttributes.requiredDocuments.length} required docs`}
            onEdit={() => startEdit(i)}
            onRemove={() => setOffers(o => o.filter((_, idx) => idx !== i))}
          />
        )
      })}

      {drafting ? (
        <AddPanel title="Offer" isEditing={editIdx !== null} addLabel="Add Offer" onAdd={handleAdd} onCancel={handleCancel} canAdd={draft.id.trim().length > 0} fieldError={fieldError}>
          <div className="flex justify-end -mt-2 -mb-2">
            <AutoFillBtn onClick={autoFill} />
          </div>
          <div className="space-y-4">
            <div><Label required>Offer ID</Label><Input value={draft.id} onChange={e => set('id', e.target.value)} placeholder="OFFER-MOTOR-COMP-Z3-4W-001" /></div>

            <div>
              <Label required>Linked Resources</Label>
              {resources.length === 0 ? (
                <p className="text-xs text-slate-400 italic px-1">No resources created yet — go back to Step 3.</p>
              ) : (
                <div className="space-y-2 mt-1">
                  {resources.map(res => (
                    <label key={res.id} className={`flex items-center gap-2 px-3 py-2 rounded-xl border cursor-pointer text-sm transition-all ${draft.resourceIds.includes(res.id) ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-slate-200 text-slate-600 hover:bg-slate-50'}`}>
                      <input type="checkbox" checked={draft.resourceIds.includes(res.id)} onChange={() => toggleResource(res.id)} className="accent-blue-600" />
                      <div className="min-w-0">
                        <p className="font-medium truncate">{res.descriptor.name || res.id}</p>
                        <p className="text-xs opacity-70 truncate">{res.id}</p>
                      </div>
                    </label>
                  ))}
                </div>
              )}
            </div>

            <div>
              <Label>Coverage Inclusions</Label>
              <p className="text-xs text-slate-400 mb-2">Subset of the linked resource's inclusions.</p>
              <CheckGroup options={availableInclusions} selected={oa.inclusions} onChange={v => set('offerAttributes.inclusions', v)} />
            </div>

            <SectionHead>Additional Terms</SectionHead>
            <StringListEditor label="Special Exclusions" items={oa.specialExclusions} onChange={v => set('offerAttributes.specialExclusions', v)} placeholder="e.g. Kendaraan dengan modifikasi struktural" />
            <StringListEditor label="Required Documents" items={oa.requiredDocuments} onChange={v => set('offerAttributes.requiredDocuments', v)} placeholder="e.g. KTP / e-KTP pengemudi utama" />

            <SectionHead>Policy IRIs</SectionHead>
            {[['claimPolicy', 'Claim Policy'], ['cancellationPolicy', 'Cancellation Policy'], ['renewalPolicy', 'Renewal Policy'], ['endorsementPolicy', 'Endorsement Policy'], ['grievanceSlaPolicy', 'Grievance SLA Policy']].map(([key, label]) => (
              <div key={key}>
                <Label>{label} IRI</Label>
                <Input value={oa[key]} onChange={e => set(`offerAttributes.${key}`, e.target.value)} placeholder={`ion://policy/insurance/${key.replace('Policy', '').toLowerCase()}`} />
              </div>
            ))}
          </div>
        </AddPanel>
      ) : (
        <AddMoreButton label="Add Offer" onClick={() => { setDraft(blankOffer()); setDrafting(true); setEditIdx(null) }} />
      )}
    </StepShell>
  )
}

// ─── Step 5: Review & Publish ─────────────────────────────────────────────────

function ReviewStep({ providers, catalogs, resources, offers }) {
  return (
    <div className="space-y-5">
      <h2 className="text-lg font-semibold text-slate-900">Review & Publish</h2>

      {/* Structure tree */}
      {providers.map(prov => {
        const provCatalogs = catalogs.filter(c => c.providerId === prov.id)
        const provResources = resources.filter(r => r.providerId === prov.id || resources.every(r2 => !r2.providerId))
        return (
          <div key={prov.id} className="border border-slate-200 rounded-2xl overflow-hidden">
            {/* Provider header */}
            <div className="bg-slate-800 text-white px-4 py-3 flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold">{prov.descriptor.name}</p>
                <p className="text-xs text-slate-400 mt-0.5">{prov.id} · OJK: {prov.providerAttributes.ojkLicenseNumber}</p>
              </div>
              <span className="text-xs px-2 py-0.5 rounded-full bg-slate-600 text-slate-300">Provider</span>
            </div>

            {/* Catalogs under this provider */}
            <div className="p-3 space-y-2 bg-white">
              {provCatalogs.length === 0 && <p className="text-xs text-amber-600 px-2">⚠ No catalogs linked to this provider</p>}
              {provCatalogs.map(cat => (
                <div key={cat.id} className="ml-3 border-l-2 border-blue-200 pl-3">
                  <div className="flex items-center gap-2 py-1">
                    <span className="text-xs font-medium text-blue-700">📁 {cat.descriptor.name}</span>
                    <span className="text-xs text-slate-400">{cat.id} · v{cat.version}</span>
                  </div>
                </div>
              ))}
            </div>

            {/* Resources under this provider */}
            {provResources.length > 0 && (
              <div className="px-3 pb-3 bg-white border-t border-slate-100 space-y-2 pt-2">
                {provResources.map(res => {
                  const resOffers = offers.filter(o => o.resourceIds.includes(res.id))
                  return (
                    <div key={res.id} className="ml-3 border-l-2 border-emerald-200 pl-3">
                      <div className="py-1">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-medium text-emerald-700">📦 {res.descriptor.name || res.id}</span>
                          <span className="text-xs text-slate-400">{res.resourceAttributes.productType} · {res.resourceAttributes.tariffZone}</span>
                        </div>
                        {resOffers.map(offer => (
                          <div key={offer.id} className="ml-3 border-l border-violet-200 pl-2 mt-1">
                            <span className="text-xs text-violet-600">🏷 {offer.id} — {offer.offerAttributes.inclusions.length} inclusions · {offer.offerAttributes.requiredDocuments.length} docs</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )
      })}

      {/* Summary row */}
      <div className="grid grid-cols-4 gap-3">
        {[
          ['Providers', providers.length, 'slate'],
          ['Catalogs', catalogs.length, 'blue'],
          ['Resources', resources.length, 'emerald'],
          ['Offers', offers.length, 'violet'],
        ].map(([label, count, color]) => (
          <div key={label} className={`bg-${color}-50 border border-${color}-200 rounded-xl p-3 text-center`}>
            <p className={`text-2xl font-bold text-${color}-700`}>{count}</p>
            <p className={`text-xs text-${color}-600 mt-0.5`}>{label}</p>
          </div>
        ))}
      </div>

      <div className="bg-amber-50 border border-amber-200 rounded-xl px-4 py-3 text-xs text-amber-800">
        Publishing will save all data to the database and broadcast catalogs to the Beckn network via beckn-onix.
      </div>
    </div>
  )
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function PublishPage() {
  const { i18n } = useTranslation()
  const lang = i18n.language === 'id' ? 'id' : 'en'

  const [step, setStep] = useState(1)
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState('')

  const [providers, setProviders] = useState([])
  const [catalogs, setCatalogs] = useState([])
  const [resources, setResources] = useState([])
  const [offers, setOffers] = useState([])

  const STEPS = ['Providers', 'Catalogs', 'Resources', 'Offers', 'Publish']

  function canAdvance() {
    if (step === 1) return providers.length > 0
    if (step === 2) return catalogs.length > 0
    if (step === 3) return resources.length > 0
    if (step === 4) return offers.length > 0
    return true
  }

  async function handlePublish() {
    setLoading(true)
    setError('')
    try {
      // Save providers to DB, build ID map (wizard ID → DB row ID)
      const providerDbIdMap = {}
      for (const prov of providers) {
        const res = await client.post('/v1/providers', {
          name: prov.descriptor.name,
          descriptor: prov.descriptor,
          locations: prov.availableAt || [],
          provider_attributes: prov.providerAttributes,
        })
        providerDbIdMap[prov.id] = res.data?.id
      }

      // Create resources (shared across catalogs)
      const resourceIdMap = {}
      for (const res of resources) {
        const ra = res.resourceAttributes
        const linkedProvider = providers.find(p => p.id === res.providerId) || providers[0]
        const resRes = await client.post('/v1/resources', {
          product_type: ra.productType,
          vehicle_type: ra.vehicleType,
          ojk_product_code: ra.ojkProductCode,
          resource_attributes: {
            '@context': 'https://schema.ion.id/finance/insurance-resource/v1/context.jsonld',
            '@type': 'ion:InsuranceProduct',
            ...ra,
            descriptor: res.descriptor,
            category: res.category,
            provider: linkedProvider ? { id: linkedProvider.id, descriptor: linkedProvider.descriptor, providerAttributes: linkedProvider.providerAttributes } : undefined,
          },
        })
        resourceIdMap[res.id] = resRes.data?.id || resRes.data?.resource_id
      }

      // Create offers
      for (const offer of offers) {
        const firstResId = offer.resourceIds[0]
        const linkedResource = resources.find(r => r.id === firstResId)
        const dbResourceId = firstResId ? resourceIdMap[firstResId] : null
        if (dbResourceId) {
          await client.post('/v1/offers', {
            resource_id: dbResourceId,
            tariff_zone: linkedResource?.resourceAttributes?.tariffZone || 'ZONE_3',
            premium_rate_min: Number(linkedResource?.resourceAttributes?.premiumRateRange?.min || 0),
            premium_rate_max: Number(linkedResource?.resourceAttributes?.premiumRateRange?.max || 0),
            offer_attributes: {
              '@context': 'https://schema.ion.id/finance/insurance-offer/v1/context.jsonld',
              '@type': 'ion:PolicyQuote',
              ...offer.offerAttributes,
              offerId: offer.id,
              resourceIds: offer.resourceIds,
            },
          })
        }
      }

      // Create and publish each catalog (with its linked provider embedded)
      for (const cat of catalogs) {
        const linkedProvider = providers.find(p => p.id === cat.providerId)
        const dbProviderID = providerDbIdMap[cat.providerId] || null
        const catRes = await client.post('/v1/catalogs', {
          name: cat.descriptor.name,
          version: cat.version,
          descriptor: cat.descriptor,
          provider: linkedProvider,
          provider_id: dbProviderID,
        })
        const catalogId = catRes.data?.id || catRes.data?.catalog_id
        await client.post('/v1/catalog/publish', { catalog_id: catalogId })
      }

      setSuccess(true)
    } catch (err) {
      setError(err.response?.data?.detail || err.response?.data?.message || err.message || 'An error occurred.')
    } finally {
      setLoading(false)
    }
  }

  function resetAll() {
    setSuccess(false); setStep(1); setError('')
    setProviders([]); setCatalogs([]); setResources([]); setOffers([])
  }

  if (success) {
    return (
      <div className="flex flex-col items-center justify-center py-24">
        <motion.div initial={{ scale: 0.7, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} className="flex flex-col items-center gap-4">
          <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center">
            <CheckCircle size={40} className="text-green-500" />
          </div>
          <h2 className="text-2xl font-bold text-slate-900">Published successfully!</h2>
          <div className="grid grid-cols-4 gap-3 mt-1">
            {[['Providers', providers.length], ['Catalogs', catalogs.length], ['Resources', resources.length], ['Offers', offers.length]].map(([l, n]) => (
              <div key={l} className="text-center bg-slate-50 rounded-xl px-4 py-2">
                <p className="text-xl font-bold text-slate-900">{n}</p>
                <p className="text-xs text-slate-500">{l}</p>
              </div>
            ))}
          </div>
          <button onClick={resetAll} className="text-blue-600 hover:underline text-sm mt-2">Publish Another →</button>
        </motion.div>
      </div>
    )
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-900 mb-6">Publish Catalog</h1>
      <div className="max-w-2xl">
        <StepIndicator current={step} total={5} labels={STEPS} />

        <AnimatePresence mode="wait">
          <motion.div key={step} initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }} transition={{ duration: 0.2 }}
            className="bg-white rounded-2xl shadow-sm border border-slate-100 p-6">

            {step === 1 && <ProvidersStep providers={providers} setProviders={setProviders} lang={lang} />}
            {step === 2 && <CatalogsStep catalogs={catalogs} setCatalogs={setCatalogs} providers={providers} lang={lang} />}
            {step === 3 && <ResourcesStep resources={resources} setResources={setResources} providers={providers} lang={lang} />}
            {step === 4 && <OffersStep offers={offers} setOffers={setOffers} resources={resources} lang={lang} />}
            {step === 5 && <ReviewStep providers={providers} catalogs={catalogs} resources={resources} offers={offers} />}

            <ErrorBox detail={error} />

            <div className="flex justify-between mt-8 pt-4 border-t border-slate-100">
              <button onClick={() => setStep(s => s - 1)} disabled={step === 1}
                className="px-5 py-2.5 text-sm font-medium text-slate-600 border border-slate-200 rounded-xl hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition">
                Back
              </button>
              {step < 5 ? (
                <button onClick={() => canAdvance() && setStep(s => s + 1)} disabled={!canAdvance()}
                  className="flex items-center gap-2 px-5 py-2.5 text-sm font-semibold bg-blue-600 text-white rounded-xl hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition">
                  Next <ChevronRight size={16} />
                </button>
              ) : (
                <button onClick={handlePublish} disabled={loading}
                  className="px-5 py-2.5 text-sm font-semibold bg-green-600 text-white rounded-xl hover:bg-green-700 disabled:opacity-50 transition">
                  {loading ? 'Publishing…' : 'Publish to Network'}
                </button>
              )}
            </div>
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  )
}
