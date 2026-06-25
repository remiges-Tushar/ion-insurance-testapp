import React, { useState, useRef, useEffect } from 'react'
import QRCode from 'qrcode'
import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { motion, AnimatePresence } from 'framer-motion'
import { Shield, CheckCircle, Download, Loader2, ArrowLeft, Wand2, Star, MessageCircle } from 'lucide-react'
import { api } from '../api.js'
import { fmtDate } from '../utils/date.js'

const DEMO_VEHICLE = {
  make: 'Toyota', model: 'Avanza', year: '2022',
  idv: '210000000', plate: 'B 1234 ABC', stnk: 'K-3456789-2022',
}

const DEMO_KYC = {
  name: 'Budi Santoso', nik: '3171012345678901', dob: '1990-01-15',
  phone: '+628123456789', email: 'budi.santoso@gmail.com',
  licenseType: 'SIM_A',
  address: 'Jl. Kemang Raya No. 15, Jakarta Selatan, DKI Jakarta 12730',
  inspectionMethod: 'SELF_INSPECTION',
}


function Spinner() {
  return (
    <div className="flex flex-col items-center justify-center py-16 gap-4">
      <Loader2 className="animate-spin text-blue-600" size={40} />
      <p className="text-slate-500">Processing your request...</p>
    </div>
  )
}

function StepIndicator({ currentStep }) {
  const steps = ['Vehicle', 'Quote', 'KYC', 'Payment', 'Policy']
  return (
    <div className="flex items-center justify-center mb-8 gap-0">
      {steps.map((step, i) => (
        <React.Fragment key={i}>
          <div className={`flex flex-col items-center ${i + 1 <= currentStep ? 'text-blue-600' : 'text-gray-400'}`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold border-2 transition-colors ${
              i + 1 < currentStep ? 'bg-blue-600 border-blue-600 text-white'
              : i + 1 === currentStep ? 'border-blue-600 text-blue-600 bg-blue-50'
              : 'border-gray-300 text-gray-400'
            }`}>
              {i + 1 < currentStep ? '✓' : i + 1}
            </div>
            <span className="text-xs mt-1 hidden sm:block">{step}</span>
          </div>
          {i < steps.length - 1 && (
            <div className={`h-0.5 w-8 sm:w-16 mx-0 ${i + 1 < currentStep ? 'bg-blue-600' : 'bg-gray-200'}`} />
          )}
        </React.Fragment>
      ))}
    </div>
  )
}

// Step 1: Vehicle Details
function VehicleDetailsStep({ product, onNext }) {
  const { t } = useTranslation()
  const [form, setForm] = useState({ make: '', model: '', year: new Date().getFullYear().toString(), idv: '', plate: '', stnk: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const submitting = useRef(false)

  function handleChange(e) {
    setForm(f => ({ ...f, [e.target.name]: e.target.value }))
  }

  async function handleSubmit(e) {
    e.preventDefault()
    if (submitting.current) return
    submitting.current = true
    setLoading(true)
    setError('')
    try {
      const payload = {
        idv: parseInt(form.idv),
        year: parseInt(form.year),
        make: form.make,
        model: form.model,
        plate: form.plate,
        stnk: form.stnk,
        productId: String(product?.id || '1'),
        productType: product?.productType || product?.product_type || 'MOTOR_COMPREHENSIVE',
        tariffZone: product?.tariffZone || product?.tariff_zone || 'ZONE_3',
      }
      const res = await api.post('/api/v1/select', payload)
      onNext(res, form)
      // On success: keep loading=true and submitting=true so the spinner stays
      // visible during the AnimatePresence exit animation instead of re-showing the form
    } catch (err) {
      setError(err?.message || 'Select failed. Please try again.')
      setLoading(false)
      submitting.current = false
    }
  }

  if (loading) return <Spinner />

  return (
    <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-bold text-slate-900">{t('vehicle.title')}</h2>
        <button type="button" onClick={() => setForm(DEMO_VEHICLE)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-violet-700 bg-violet-50 border border-violet-200 rounded-lg hover:bg-violet-100 transition">
          <Wand2 size={13} /> Auto Fill
        </button>
      </div>
      {product && (
        <div className="mb-6 p-4 bg-blue-50 rounded-xl text-sm">
          <p className="font-medium text-blue-900">{product.insurer_name || product.insurerName || 'Selected Insurer'}</p>
          <p className="text-blue-700">{product.product_type || product.productType} — {product.vehicle_type || product.vehicleType}</p>
        </div>
      )}
      {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm">{error}</div>}
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">{t('vehicle.make')}</label>
            <input name="make" value={form.make} onChange={handleChange} required placeholder="Toyota"
              className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">{t('vehicle.model')}</label>
            <input name="model" value={form.model} onChange={handleChange} required placeholder="Avanza"
              className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('vehicle.year')}</label>
          <input name="year" type="number" min="2000" max="2025" value={form.year} onChange={handleChange} required
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('vehicle.idv')}</label>
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">IDR</span>
            <input name="idv" type="number" value={form.idv} onChange={handleChange} required placeholder="150000000"
              className="w-full border border-slate-300 rounded-lg pl-12 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('vehicle.plate')}</label>
          <input name="plate" value={form.plate} onChange={handleChange} required placeholder="B 1234 ABC"
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('vehicle.stnk')}</label>
          <input name="stnk" value={form.stnk} onChange={handleChange} required placeholder="STNK number"
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <button type="submit" disabled={loading}
          className="w-full bg-blue-600 text-white py-3 rounded-xl font-medium hover:bg-blue-700 transition-colors disabled:opacity-60 disabled:cursor-not-allowed">
          {t('common.next')}
        </button>
      </form>
    </motion.div>
  )
}

// Step 2: Quote Display
function QuoteStep({ quoteData, onNext, onBack }) {
  const { t } = useTranslation()
  // Beckn v2 on_select: message.contract.commitments[0].offer.offerAttributes
  const offerAttrs = quoteData?.message?.contract?.commitments?.[0]?.offer?.offerAttributes || {}

  const approvedIDV = offerAttrs.approvedIDV || offerAttrs.approvedIdv || 0
  const annualPremium = offerAttrs.annualPremiumIDR || offerAttrs.totalPremium || 0
  const breakup = offerAttrs.breakup || []
  const basePremium = breakup.find(b => b.title === 'Base Premium')?.amountIDR || 0
  const adminFee = breakup.find(b => b.title === 'Admin Fee')?.amountIDR || 0
  const stampDuty = breakup.find(b => b.title === 'Stamp Duty')?.amountIDR || 10000
  const validUntil = offerAttrs.quoteValidUntil

  return (
    <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
      <h2 className="text-xl font-bold text-slate-900 mb-6">{t('quote.title')}</h2>
      <div className="bg-gradient-to-br from-blue-600 to-indigo-700 rounded-2xl p-6 text-white mb-6">
        <p className="text-blue-100 text-sm mb-1">{t('quote.annual_premium')}</p>
        <p className="text-4xl font-bold">IDR {Number(annualPremium).toLocaleString()}</p>
        {validUntil && <p className="text-blue-200 text-xs mt-2">{t('quote.valid_until')}: {fmtDate(validUntil)}</p>}
      </div>
      <div className="bg-white border border-gray-100 rounded-xl p-5 space-y-3 mb-6 shadow-sm">
        {[
          [t('quote.approved_idv'), `IDR ${Number(approvedIDV).toLocaleString()}`],
          [t('quote.base_premium'), `IDR ${Number(basePremium).toLocaleString()}`],
          [t('quote.admin_fee'), `IDR ${Number(adminFee).toLocaleString()}`],
          [t('quote.stamp_duty'), `IDR ${Number(stampDuty).toLocaleString()}`],
        ].map(([label, value]) => (
          <div key={label} className="flex justify-between text-sm">
            <span className="text-slate-600">{label}</span>
            <span className="font-medium text-slate-900">{value}</span>
          </div>
        ))}
        <div className="pt-3 border-t border-gray-100 flex justify-between text-sm font-bold">
          <span className="text-slate-900">{t('quote.annual_premium')}</span>
          <span className="text-blue-700">IDR {Number(annualPremium).toLocaleString()}</span>
        </div>
      </div>
      <div className="flex gap-3">
        <button onClick={onBack} className="flex-1 border border-gray-300 text-slate-600 py-3 rounded-xl font-medium hover:bg-gray-50">{t('common.back')}</button>
        <button onClick={onNext} className="flex-1 bg-blue-600 text-white py-3 rounded-xl font-medium hover:bg-blue-700 transition-colors">{t('quote.apply')}</button>
      </div>
    </motion.div>
  )
}

// Step 3: KYC
function KYCStep({ onNext, onBack, txnId, vehicleForm }) {
  const { t } = useTranslation()
  const [form, setForm] = useState({
    name: '', nik: '', dob: '', phone: '', email: '',
    licenseType: 'SIM_A', address: '', inspectionMethod: 'SELF_INSPECTION'
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [nikError, setNikError] = useState('')
  const submitting = useRef(false)

  function handleChange(e) {
    setForm(f => ({ ...f, [e.target.name]: e.target.value }))
    if (e.target.name === 'nik') {
      setNikError(e.target.value.length !== 16 ? 'NIK must be exactly 16 digits' : '')
    }
  }

  async function handleSubmit(e) {
    e.preventDefault()
    if (form.nik.length !== 16) { setNikError('NIK must be exactly 16 digits'); return }
    if (submitting.current) return
    submitting.current = true
    setLoading(true)
    setError('')
    try {
      const res = await api.post('/api/v1/init', {
        ...form,
        transaction_id: txnId,
        plate: vehicleForm?.plate || '',
        idv: vehicleForm?.idv ? parseInt(vehicleForm.idv) : 0,
      })
      onNext(res)
    } catch (err) {
      setError(err?.message || 'Failed to submit KYC. Please try again.')
      setLoading(false)
      submitting.current = false
    }
  }

  if (loading) return <Spinner />

  return (
    <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-bold text-slate-900">{t('kyc.title')}</h2>
        <button type="button" onClick={() => { setForm(DEMO_KYC); setNikError('') }}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-violet-700 bg-violet-50 border border-violet-200 rounded-lg hover:bg-violet-100 transition">
          <Wand2 size={13} /> Auto Fill
        </button>
      </div>
      {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm">{error}</div>}
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.name')}</label>
          <input name="name" value={form.name} onChange={handleChange} required
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.nik')}</label>
          <input name="nik" value={form.nik} onChange={handleChange} required maxLength={16} pattern="\d{16}"
            className={`w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${nikError ? 'border-red-400' : 'border-slate-300'}`} />
          {nikError && <p className="text-red-500 text-xs mt-1">{nikError}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.dob')}</label>
          <input name="dob" type="date" value={form.dob} onChange={handleChange} required
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.phone')}</label>
          <input name="phone" type="tel" value={form.phone} onChange={handleChange} required placeholder="+62..."
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.email')}</label>
          <input name="email" type="email" value={form.email} onChange={handleChange} required
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.license_type')}</label>
          <select name="licenseType" value={form.licenseType} onChange={handleChange}
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option value="SIM_A">SIM A</option>
            <option value="SIM_B">SIM B</option>
            <option value="SIM_C">SIM C</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">{t('kyc.address')}</label>
          <textarea name="address" value={form.address} onChange={handleChange} required rows={3}
            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">{t('kyc.inspection')}</label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 cursor-pointer text-sm">
              <input type="radio" name="inspectionMethod" value="SELF_INSPECTION"
                checked={form.inspectionMethod === 'SELF_INSPECTION'} onChange={handleChange} className="text-blue-600" />
              {t('kyc.self_inspection')}
            </label>
            <label className="flex items-center gap-2 cursor-pointer text-sm">
              <input type="radio" name="inspectionMethod" value="SURVEYOR_VISIT"
                checked={form.inspectionMethod === 'SURVEYOR_VISIT'} onChange={handleChange} className="text-blue-600" />
              {t('kyc.surveyor')}
            </label>
          </div>
        </div>
        <div className="flex gap-3">
          <button type="button" onClick={onBack} className="flex-1 border border-gray-300 text-slate-600 py-3 rounded-xl font-medium hover:bg-gray-50">{t('common.back')}</button>
          <button type="submit" disabled={loading} className="flex-1 bg-blue-600 text-white py-3 rounded-xl font-medium hover:bg-blue-700 transition-colors disabled:opacity-60 disabled:cursor-not-allowed">{t('common.next')}</button>
        </div>
      </form>
    </motion.div>
  )
}

function QRCanvas({ qrString }) {
  const canvasRef = useRef(null)
  useEffect(() => {
    if (canvasRef.current && qrString) {
      QRCode.toCanvas(canvasRef.current, qrString, { width: 220, margin: 2 })
    }
  }, [qrString])
  if (!qrString) return <p className="text-slate-400 text-sm text-center py-8">QR code not available</p>
  return <canvas ref={canvasRef} className="mx-auto rounded-lg" />
}

// Step 4: Payment
function PaymentStep({ initData, onNext, onBack, txnId, annualPremium }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [simulating, setSimulating] = useState(false)
  const [seamStatus, setSeamStatus] = useState(null) // null | {policy_status, payment_received, seam_stage, doku_invoice_number}
  const [error, setError] = useState('')
  const [tab, setTab] = useState('VA')
  const submitting = useRef(false)

  // VA details are in considerationAttributes; commitmentAttributes is the fallback
  const considerationAttrs = initData?.message?.contract?.commitments?.[0]?.considerationAttributes || {}
  const commitmentAttrs = initData?.message?.contract?.commitments?.[0]?.commitmentAttributes || {}
  const totalPremium = considerationAttrs.totalAmountIDR || annualPremium || 0
  const virtualAccount = considerationAttrs.virtualAccount || commitmentAttrs.virtualAccount || '—'
  const bankCode = considerationAttrs.bankCode || commitmentAttrs.bankCode || ''
  const qrisString = considerationAttrs.qrisString || ''

  async function handleSimulate() {
    setSimulating(true)
    setError('')
    try {
      await api.post(`/webhook/simulate-payment?txn_id=${txnId}&method=${tab}`)
      // Fetch SEAM status to show the hold proof
      const status = await api.get(`/api/v1/payment-status?txn_id=${txnId}`)
      setSeamStatus(status)
    } catch (err) {
      setError(err?.message || 'Simulation failed.')
    } finally {
      setSimulating(false)
    }
  }

  async function handlePaid() {
    if (submitting.current) return
    submitting.current = true
    setLoading(true)
    setError('')
    try {
      const isQRIS = tab === 'QRIS'
      const res = await api.post('/api/v1/confirm', {
        transaction_id: txnId,
        paymentMethod: isQRIS ? 'QRIS' : 'VIRTUAL_ACCOUNT',
        paymentRef: isQRIS ? `QRIS-${txnId}` : `VA-${virtualAccount}`,
        amount: totalPremium,
      })
      onNext(res)
    } catch (err) {
      setError(err?.message || 'Failed to confirm payment. Please try again.')
      setLoading(false)
      submitting.current = false
    }
  }

  const fundsHeld = seamStatus?.payment_received && seamStatus?.policy_status !== 'ACTIVE'

  if (loading) return <Spinner />

  return (
    <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
      <h2 className="text-xl font-bold text-slate-900 mb-6">{t('payment.title')}</h2>
      {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm">{error}</div>}

      {/* SEAM stage indicator — visible after simulate */}
      {seamStatus && (
        <div className={`mb-4 p-4 rounded-xl border ${fundsHeld ? 'bg-orange-50 border-orange-200' : 'bg-green-50 border-green-200'}`}>
          <p className="text-xs font-bold uppercase tracking-wide mb-1 ${fundsHeld ? 'text-orange-700' : 'text-green-700'}">
            SEAM — {seamStatus.seam_stage}
          </p>
          <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs mt-2">
            <span className="text-slate-500">Policy Status</span>
            <span className={`font-semibold ${seamStatus.policy_status === 'ACTIVE' ? 'text-green-700' : 'text-orange-700'}`}>{seamStatus.policy_status}</span>
            <span className="text-slate-500">Funds at DOKU</span>
            <span className={`font-semibold ${fundsHeld ? 'text-orange-700' : 'text-green-700'}`}>{fundsHeld ? 'HELD (not settled yet)' : seamStatus.policy_status === 'ACTIVE' ? 'RELEASED' : '—'}</span>
            <span className="text-slate-500">Invoice</span>
            <span className="font-mono text-slate-700">{seamStatus.doku_invoice_number || '—'}</span>
          </div>
          {fundsHeld && (
            <p className="text-xs text-orange-600 mt-2 font-medium">
              Proof: policy is still PENDING — DOKU is holding the funds. Click "Release &amp; Issue Policy" below to trigger Stage 3.
            </p>
          )}
        </div>
      )}

      <div className="bg-white border border-gray-100 rounded-xl p-6 shadow-sm mb-6 space-y-4">
        <div>
          <p className="text-sm text-slate-500 mb-1">{t('payment.total')}</p>
          <p className="text-3xl font-bold text-slate-900">IDR {Number(totalPremium).toLocaleString()}</p>
        </div>
        {/* Payment method tabs */}
        <div className="flex rounded-lg border border-gray-200 overflow-hidden">
          <button
            onClick={() => { setTab('VA'); setSeamStatus(null) }}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${tab === 'VA' ? 'bg-blue-600 text-white' : 'bg-white text-slate-600 hover:bg-gray-50'}`}
          >
            Virtual Account
          </button>
          <button
            onClick={() => { setTab('QRIS'); setSeamStatus(null) }}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${tab === 'QRIS' ? 'bg-blue-600 text-white' : 'bg-white text-slate-600 hover:bg-gray-50'}`}
          >
            QRIS
          </button>
        </div>

        {tab === 'VA' && (
          <div className="border-t border-gray-100 pt-4">
            <p className="text-sm text-slate-500 mb-2">{t('payment.virtual_account')}</p>
            {bankCode && (
              <p className="text-sm font-semibold text-slate-700 mb-1 text-center">Bank {bankCode}</p>
            )}
            <div className="bg-gray-50 rounded-lg px-4 py-3 font-mono text-lg font-bold tracking-wider text-slate-900 text-center">
              {virtualAccount}
            </div>
            <p className="text-xs text-slate-400 mt-2 text-center">Transfer the exact amount to this virtual account</p>
          </div>
        )}

        {tab === 'QRIS' && (
          <div className="border-t border-gray-100 pt-4 flex flex-col items-center gap-3">
            {qrisString ? (
              <>
                <p className="text-sm text-slate-500">Scan with your mobile banking or e-wallet app</p>
                <QRCanvas qrString={qrisString} />
                <p className="text-xs text-slate-400 text-center">QRIS — IDR {Number(totalPremium).toLocaleString()}</p>
              </>
            ) : (
              <div className="py-6 text-center">
                <p className="text-sm text-slate-500 font-medium">DOKU QRIS — Coming Soon</p>
                <p className="text-xs text-slate-400 mt-1">Use Virtual Account for now</p>
              </div>
            )}
          </div>
        )}

        {/* Sandbox test helper */}
        <div className="border-t border-dashed border-amber-200 pt-4">
          <p className="text-xs text-amber-600 font-medium mb-1 text-center">⚡ DOKU Sandbox — SEAM Test</p>
          <p className="text-xs text-slate-400 text-center mb-2">Step 1: Simulate payment → funds held. Step 2: Confirm → release &amp; issue policy.</p>
          <button
            onClick={handleSimulate}
            disabled={simulating}
            className="w-full py-2 text-sm border border-amber-400 text-amber-700 rounded-lg hover:bg-amber-50 transition-colors disabled:opacity-50"
          >
            {simulating ? 'Simulating...' : fundsHeld ? 'Re-simulate Payment' : 'Simulate DOKU VA Payment (SEAM Hold)'}
          </button>
        </div>
      </div>
      <div className="flex gap-3">
        <button onClick={onBack} className="flex-1 border border-gray-300 text-slate-600 py-3 rounded-xl font-medium hover:bg-gray-50">{t('common.back')}</button>
        <button
          onClick={handlePaid}
          disabled={loading || !seamStatus?.payment_received}
          className={`flex-1 py-3 rounded-xl font-medium transition-colors disabled:opacity-60 disabled:cursor-not-allowed ${fundsHeld ? 'bg-orange-500 hover:bg-orange-600 text-white' : 'bg-green-600 hover:bg-green-700 text-white'}`}
        >
          {fundsHeld ? 'Release Funds & Issue Policy (SEAM Stage 3)' : t('payment.paid')}
        </button>
      </div>
    </motion.div>
  )
}

// Step 5: Policy Issued
function PolicyIssuedStep({ confirmData, txnId }) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  // Beckn v2 on_confirm: message.contract.commitments[0].commitmentAttributes
  const attrs = confirmData?.message?.contract?.commitments?.[0]?.commitmentAttributes || {}
  const policyNumber = attrs.policyNumber || '—'
  const certificateUrl = attrs.certificateUrl
  const coverageStart = attrs.coverageStart || attrs.coverageStartDate
  const coverageEnd = attrs.coverageEnd || attrs.coverageEndDate

  // Rating state
  const [rating, setRating] = useState(0)
  const [hoverRating, setHoverRating] = useState(0)
  const [feedback, setFeedback] = useState('')
  const [ratingSubmitted, setRatingSubmitted] = useState(false)
  const [ratingLoading, setRatingLoading] = useState(false)
  const [ratingError, setRatingError] = useState('')

  // Support state
  const [showSupport, setShowSupport] = useState(false)
  const [supportDesc, setSupportDesc] = useState('')
  const [supportSubmitted, setSupportSubmitted] = useState(false)
  const [supportLoading, setSupportLoading] = useState(false)
  const [supportError, setSupportError] = useState('')

  async function handleSubmitRating() {
    if (!rating) return
    setRatingLoading(true)
    setRatingError('')
    try {
      await api.post('/api/v1/rate', {
        transaction_id: txnId,
        score: rating,
        feedback: feedback.trim() || undefined,
      })
      setRatingSubmitted(true)
    } catch {
      setRatingError('Failed to submit rating. Please try again.')
    } finally {
      setRatingLoading(false)
    }
  }

  async function handleSubmitSupport() {
    if (!supportDesc.trim()) return
    setSupportLoading(true)
    setSupportError('')
    try {
      await api.post('/api/v1/support', {
        transaction_id: txnId,
        description: supportDesc.trim(),
      })
      setSupportSubmitted(true)
    } catch {
      setSupportError('Failed to submit support request. Please try again.')
    } finally {
      setSupportLoading(false)
    }
  }

  return (
    <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} className="text-center">
      <div className="flex justify-center mb-6">
        <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center">
          <CheckCircle className="text-green-600" size={48} />
        </div>
      </div>
      <h2 className="text-2xl font-bold text-slate-900 mb-2">{t('policy.title')}</h2>
      <p className="text-slate-500 mb-6">Your motor insurance policy has been issued.</p>

      {/* Policy details */}
      <div className="bg-white border border-gray-100 rounded-xl p-5 shadow-sm text-left space-y-3 mb-4">
        <div className="flex justify-between items-center">
          <span className="text-slate-500 text-sm">{t('policy.number')}</span>
          <span className="font-bold text-slate-900 font-mono text-sm">{policyNumber}</span>
        </div>
        {coverageStart && (
          <div className="flex justify-between items-center">
            <span className="text-slate-500 text-sm">{t('policy.coverage_start')}</span>
            <span className="font-medium text-slate-900 text-sm">{fmtDate(coverageStart)}</span>
          </div>
        )}
        {coverageEnd && (
          <div className="flex justify-between items-center">
            <span className="text-slate-500 text-sm">{t('policy.coverage_end')}</span>
            <span className="font-medium text-slate-900 text-sm">{fmtDate(coverageEnd)}</span>
          </div>
        )}
      </div>

      {certificateUrl && (
        <a href={certificateUrl} target="_blank" rel="noopener noreferrer"
          className="flex items-center justify-center gap-2 w-full mb-4 px-6 py-2.5 border-2 border-blue-600 text-blue-600 rounded-xl font-medium hover:bg-blue-50 transition-colors text-sm">
          <Download size={16} />
          {t('policy.certificate')}
        </a>
      )}

      {/* Rating section */}
      <div className="bg-amber-50 border border-amber-100 rounded-xl p-5 mb-4 text-left">
        <div className="flex items-center gap-2 mb-3">
          <Star className="text-amber-500" size={18} />
          <span className="font-semibold text-slate-800 text-sm">Rate your experience</span>
        </div>

        {ratingSubmitted ? (
          <div className="text-center py-2">
            <p className="text-green-700 font-medium text-sm">Thank you for your rating!</p>
          </div>
        ) : (
          <>
            {/* Star selector */}
            <div className="flex justify-center gap-2 mb-3">
              {[1, 2, 3, 4, 5].map(star => (
                <button key={star}
                  onClick={() => setRating(star)}
                  onMouseEnter={() => setHoverRating(star)}
                  onMouseLeave={() => setHoverRating(0)}
                  className="focus:outline-none transition-transform hover:scale-110">
                  <Star
                    size={32}
                    className={star <= (hoverRating || rating)
                      ? 'text-amber-400 fill-amber-400'
                      : 'text-slate-300'}
                  />
                </button>
              ))}
            </div>
            {rating > 0 && (
              <p className="text-center text-xs text-slate-500 mb-3">
                {['', 'Poor', 'Fair', 'Good', 'Very Good', 'Excellent'][rating]}
              </p>
            )}

            {/* Feedback textarea */}
            <textarea
              value={feedback}
              onChange={e => setFeedback(e.target.value)}
              placeholder="Share your feedback (optional)..."
              rows={2}
              className="w-full border border-amber-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400 bg-white resize-none mb-3"
            />

            {ratingError && <p className="text-red-600 text-xs mb-2">{ratingError}</p>}

            <button
              onClick={handleSubmitRating}
              disabled={!rating || ratingLoading}
              className="w-full bg-amber-500 text-white py-2 rounded-lg font-medium hover:bg-amber-600 transition-colors disabled:opacity-50 text-sm">
              {ratingLoading ? 'Submitting...' : 'Submit Rating'}
            </button>
          </>
        )}
      </div>

      {/* Support section */}
      <div className="bg-slate-50 border border-slate-200 rounded-xl p-5 mb-4 text-left">
        <button
          onClick={() => setShowSupport(s => !s)}
          className="flex items-center justify-between w-full">
          <div className="flex items-center gap-2">
            <MessageCircle className="text-slate-500" size={18} />
            <span className="font-semibold text-slate-800 text-sm">Need support?</span>
          </div>
          <span className="text-slate-400 text-xs">{showSupport ? '▲' : '▼'}</span>
        </button>

        {showSupport && (
          <div className="mt-3">
            {supportSubmitted ? (
              <p className="text-green-700 font-medium text-sm text-center py-1">
                Support ticket created. Our team will contact you shortly.
              </p>
            ) : (
              <>
                <textarea
                  value={supportDesc}
                  onChange={e => setSupportDesc(e.target.value)}
                  placeholder="Describe your issue or question..."
                  rows={3}
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none mb-3"
                />
                {supportError && <p className="text-red-600 text-xs mb-2">{supportError}</p>}
                <button
                  onClick={handleSubmitSupport}
                  disabled={!supportDesc.trim() || supportLoading}
                  className="w-full bg-slate-700 text-white py-2 rounded-lg font-medium hover:bg-slate-800 transition-colors disabled:opacity-50 text-sm">
                  {supportLoading ? 'Submitting...' : 'Submit Support Request'}
                </button>
              </>
            )}
          </div>
        )}
      </div>

      <button onClick={() => navigate('/history')}
        className="w-full bg-blue-600 text-white py-3 rounded-xl font-medium hover:bg-blue-700 transition-colors">
        {t('policy.history')}
      </button>
    </motion.div>
  )
}

// Main flow page
export default function PolicyFlowPage() {
  const { state } = useLocation()
  const [step, setStep] = useState(1)
  const [quoteData, setQuoteData] = useState(null)
  const [initData, setInitData] = useState(null)
  const [confirmData, setConfirmData] = useState(null)
  const [actualTxnId, setActualTxnId] = useState(null)
  const [vehicleForm, setVehicleForm] = useState(null)
  const navigate = useNavigate()

  // HeroPage passes state.selectedItem
  const product = state?.selectedItem || state?.product || null

  const displayTxnId = actualTxnId || '—'
  const annualPremium = quoteData?.message?.contract?.commitments?.[0]?.offer?.offerAttributes?.annualPremiumIDR || 0

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <header className="bg-white shadow-sm">
        <div className="max-w-2xl mx-auto px-4 py-4 flex items-center gap-3">
          <button onClick={() => navigate('/')} className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-2">
            <Shield className="text-blue-600" size={22} />
            <span className="font-bold text-slate-900">MotorInsure</span>
          </div>
          <span className="text-slate-400 text-sm ml-auto font-mono text-xs truncate max-w-[160px]">{displayTxnId}</span>
        </div>
      </header>

      <div className="max-w-2xl mx-auto px-4 py-8">
        <StepIndicator currentStep={step} />
        <div className="bg-white rounded-2xl shadow-lg p-6 md:p-8">
          <AnimatePresence mode="wait">
            {step === 1 && (
              <VehicleDetailsStep key="step1" product={product}
                onNext={(res, form) => {
                  setQuoteData(res)
                  setVehicleForm(form)
                  setActualTxnId(res?.context?.transaction_id || null)
                  setStep(2)
                }} />
            )}
            {step === 2 && (
              <QuoteStep key="step2" quoteData={quoteData}
                onNext={() => setStep(3)} onBack={() => setStep(1)} />
            )}
            {step === 3 && (
              <KYCStep key="step3" txnId={actualTxnId} vehicleForm={vehicleForm}
                onNext={(res) => { setInitData(res); setStep(4) }} onBack={() => setStep(2)} />
            )}
            {step === 4 && (
              <PaymentStep key="step4" initData={initData} txnId={actualTxnId} annualPremium={annualPremium}
                onNext={(res) => { setConfirmData(res); setStep(5) }} onBack={() => setStep(3)} />
            )}
            {step === 5 && (
              <PolicyIssuedStep key="step5" confirmData={confirmData} txnId={actualTxnId} />
            )}
          </AnimatePresence>
        </div>
      </div>
    </div>
  )
}
