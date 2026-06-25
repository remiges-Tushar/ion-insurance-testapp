import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Shield, ArrowLeft, X, Download, Calendar, DollarSign, FileText, TrendingUp, CheckCircle, Clock, AlertCircle, CreditCard, RefreshCw, Banknote } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../api.js'
import { fmtDate } from '../utils/date.js'
import StatusBadge from '../components/StatusBadge.jsx'
import SeamStageBar from '../components/SeamStageBar.jsx'

// ─── Status meta for policies ────────────────────────────────────────────────

const STATUS_META = {
  ACTIVE:           { icon: CheckCircle, color: 'text-emerald-500', bg: 'bg-emerald-50', border: 'border-emerald-200' },
  PENDING_ISSUANCE: { icon: Clock,       color: 'text-amber-500',   bg: 'bg-amber-50',   border: 'border-amber-200' },
  CANCELLED:        { icon: X,           color: 'text-red-500',     bg: 'bg-red-50',     border: 'border-red-200' },
  EXPIRED:          { icon: AlertCircle, color: 'text-slate-400',   bg: 'bg-slate-50',   border: 'border-slate-200' },
  TOTAL_LOSS:       { icon: AlertCircle, color: 'text-red-600',     bg: 'bg-red-50',     border: 'border-red-200' },
}

const SEAM_META = {
  va_created:    { label: 'VA Created',     color: 'text-blue-600',    bg: 'bg-blue-50',    border: 'border-blue-200' },
  payment_held:  { label: 'Payment Held',   color: 'text-amber-600',   bg: 'bg-amber-50',   border: 'border-amber-200' },
  policy_issued: { label: 'Policy Issued',  color: 'text-indigo-600',  bg: 'bg-indigo-50',  border: 'border-indigo-200' },
  reconciling:   { label: 'Reconciling',    color: 'text-violet-600',  bg: 'bg-violet-50',  border: 'border-violet-200' },
  settled:       { label: 'Settled',        color: 'text-emerald-600', bg: 'bg-emerald-50', border: 'border-emerald-200' },
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function fmtIDR(n) {
  if (!n || n === 0) return '—'
  return 'IDR ' + Number(n).toLocaleString('id-ID')
}

function fmtDateShort(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' })
}

// ─── Coverage progress pill ───────────────────────────────────────────────────

function CoveragePill({ start, end }) {
  if (!start || !end) return null
  const now = Date.now()
  const s = new Date(start).getTime()
  const e = new Date(end).getTime()
  const total = e - s
  const elapsed = Math.max(0, Math.min(now - s, total))
  const pct = total > 0 ? Math.round((elapsed / total) * 100) : 0
  const daysLeft = Math.max(0, Math.round((e - now) / 86400000))
  const isActive = now >= s && now <= e
  return (
    <div className="mt-3 pt-3 border-t border-gray-50">
      <div className="flex justify-between text-xs text-slate-400 mb-1.5">
        <span>{fmtDate(start)}</span>
        <span className={isActive ? 'text-emerald-600 font-semibold' : 'text-slate-400'}>
          {isActive ? `${daysLeft}d left` : now > e ? 'Expired' : 'Not started'}
        </span>
        <span>{fmtDate(end)}</span>
      </div>
      <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${isActive ? 'bg-emerald-400' : 'bg-slate-300'}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

// ─── Order detail slide-in panel ─────────────────────────────────────────────

function OrderDetailPanel({ order, onClose }) {
  if (!order) return null
  const seam = SEAM_META[order.seam_stage] || SEAM_META.va_created

  return (
    <AnimatePresence>
      <motion.div
        key="overlay"
        initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/40 z-40" onClick={onClose}
      />
      <motion.div
        key="panel"
        initial={{ x: '100%' }} animate={{ x: 0 }} exit={{ x: '100%' }}
        transition={{ type: 'spring', damping: 25 }}
        className="fixed right-0 top-0 h-full w-full sm:w-[440px] bg-white shadow-2xl z-50 overflow-y-auto"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-indigo-600 to-violet-600">
          <div>
            <p className="text-indigo-200 text-xs font-medium mb-0.5">Transaction</p>
            <h2 className="text-white font-bold text-sm font-mono break-all">{order.transaction_id}</h2>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-white/10 rounded-lg text-white/70 hover:text-white transition-colors shrink-0 ml-3">
            <X size={18} />
          </button>
        </div>

        <div className="p-6 space-y-5">
          {/* SEAM stage badge */}
          <div className={`flex items-center gap-3 p-3 rounded-xl border ${seam.bg} ${seam.border}`}>
            <RefreshCw size={16} className={seam.color} />
            <div>
              <p className="text-xs text-slate-500">SEAM Stage</p>
              <p className={`font-semibold text-sm ${seam.color}`}>{seam.label}</p>
            </div>
          </div>

          {/* SEAM stage bar */}
          <div className="bg-slate-50 rounded-xl p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3">Settlement Progress</p>
            <SeamStageBar stage={order.seam_stage} />
          </div>

          {/* Payment details */}
          <div className="bg-slate-50 rounded-xl p-4 space-y-3">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide flex items-center gap-1.5">
              <CreditCard size={11} /> Payment Details
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="text-xs text-slate-400">Amount</p>
                <p className="text-sm font-bold text-slate-800">{fmtIDR(order.payment_amount)}</p>
              </div>
              <div>
                <p className="text-xs text-slate-400">VA Number</p>
                <p className="text-sm font-mono font-semibold text-slate-800">{order.doku_va_number || '—'}</p>
              </div>
            </div>
            <div>
              <p className="text-xs text-slate-400">Invoice Number</p>
              <p className="text-xs font-mono bg-white border border-slate-200 rounded-lg px-2.5 py-1.5 text-slate-600 break-all">
                {order.doku_invoice_number || '—'}
              </p>
            </div>
          </div>

          {/* Timestamps */}
          <div className="space-y-2">
            <div className="flex justify-between">
              <p className="text-xs text-slate-400">Created</p>
              <p className="text-xs text-slate-600 font-medium">{fmtDateShort(order.created_at)}</p>
            </div>
            <div className="flex justify-between">
              <p className="text-xs text-slate-400">Last Updated</p>
              <p className="text-xs text-slate-600 font-medium">{fmtDateShort(order.last_updated)}</p>
            </div>
            <div className="flex justify-between">
              <p className="text-xs text-slate-400">Latest Action</p>
              <p className="text-xs font-mono bg-slate-100 rounded px-2 py-0.5 text-slate-700">{order.latest_on_action || order.action || '—'}</p>
            </div>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  )
}

// ─── Policy detail slide-in panel ────────────────────────────────────────────

function PolicyDetailPanel({ policy, onClose }) {
  if (!policy) return null
  const meta = STATUS_META[policy.status] || STATUS_META.ACTIVE
  const StatusIcon = meta.icon

  return (
    <AnimatePresence>
      <motion.div
        key="overlay"
        initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/40 z-40" onClick={onClose}
      />
      <motion.div
        key="panel"
        initial={{ x: '100%' }} animate={{ x: 0 }} exit={{ x: '100%' }}
        transition={{ type: 'spring', damping: 25 }}
        className="fixed right-0 top-0 h-full w-full sm:w-[420px] bg-white shadow-2xl z-50 overflow-y-auto"
      >
        <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-blue-600 to-indigo-600">
          <div>
            <p className="text-blue-200 text-xs font-medium mb-0.5">Policy Number</p>
            <h2 className="text-white font-bold text-lg">{policy.policy_number || '—'}</h2>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-white/10 rounded-lg text-white/70 hover:text-white transition-colors">
            <X size={18} />
          </button>
        </div>

        <div className="p-6 space-y-5">
          <div className={`flex items-center gap-3 p-3 rounded-xl border ${meta.bg} ${meta.border}`}>
            <StatusIcon size={18} className={meta.color} />
            <div>
              <p className="text-xs text-slate-500">Policy Status</p>
              <p className={`font-semibold text-sm ${meta.color}`}>{policy.status || 'ACTIVE'}</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="bg-blue-50 rounded-xl p-4">
              <p className="text-xs text-blue-500 font-medium mb-1 flex items-center gap-1"><DollarSign size={10} /> Annual Premium</p>
              <p className="text-lg font-bold text-blue-700">{policy.annual_premium ? `IDR ${Number(policy.annual_premium).toLocaleString('id-ID')}` : '—'}</p>
            </div>
            <div className="bg-indigo-50 rounded-xl p-4">
              <p className="text-xs text-indigo-500 font-medium mb-1 flex items-center gap-1"><TrendingUp size={10} /> IDV</p>
              <p className="text-lg font-bold text-indigo-700">{policy.idv ? `IDR ${Number(policy.idv).toLocaleString('id-ID')}` : '—'}</p>
            </div>
          </div>

          <div className="bg-slate-50 rounded-xl p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3 flex items-center gap-1.5"><Calendar size={11} /> Coverage Period</p>
            <div className="flex gap-4">
              <div>
                <p className="text-xs text-slate-400">Start</p>
                <p className="text-sm font-semibold text-slate-800">{fmtDate(policy.coverage_start)}</p>
              </div>
              <div className="w-px bg-slate-200" />
              <div>
                <p className="text-xs text-slate-400">End</p>
                <p className="text-sm font-semibold text-slate-800">{fmtDate(policy.coverage_end)}</p>
              </div>
            </div>
            <CoveragePill start={policy.coverage_start} end={policy.coverage_end} />
          </div>

          <div>
            <p className="text-xs text-slate-500 mb-1">Transaction ID</p>
            <p className="text-xs font-mono bg-slate-50 border border-slate-100 rounded-lg px-3 py-2 text-slate-600 break-all">{policy.transaction_id}</p>
          </div>

          {policy.certificate_url && (
            <a
              href={policy.certificate_url} target="_blank" rel="noreferrer"
              className="flex items-center gap-3 p-4 border-2 border-dashed border-blue-200 rounded-xl hover:bg-blue-50 transition-colors group"
            >
              <Download size={18} className="text-blue-500 group-hover:text-blue-700" />
              <div>
                <p className="text-sm font-semibold text-blue-700">Download Certificate</p>
                <p className="text-xs text-slate-400">Policy certificate (PDF)</p>
              </div>
            </a>
          )}

          <div>
            <p className="text-xs text-slate-400">Issued on</p>
            <p className="text-sm text-slate-600">{fmtDate(policy.received_at)}</p>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  )
}

// ─── Summary bars ─────────────────────────────────────────────────────────────

function OrderSummaryBar({ orders }) {
  const settled = orders.filter(o => o.seam_stage === 'settled').length
  const inProgress = orders.filter(o => o.seam_stage !== 'settled').length
  const totalPaid = orders.reduce((s, o) => s + (Number(o.payment_amount) || 0), 0)

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
      {[
        { label: 'Total Orders',  value: orders.length,                     icon: FileText,  color: 'text-blue-600',    bg: 'bg-blue-50' },
        { label: 'Settled',       value: settled,                           icon: CheckCircle, color: 'text-emerald-600', bg: 'bg-emerald-50' },
        { label: 'In Progress',   value: inProgress,                        icon: Clock,     color: 'text-amber-600',   bg: 'bg-amber-50' },
        { label: 'Total Paid',    value: totalPaid ? `IDR ${(totalPaid / 1e6).toFixed(1)}M` : '—', icon: Banknote, color: 'text-violet-600', bg: 'bg-violet-50' },
      ].map(({ label, value, icon: Icon, color, bg }) => (
        <div key={label} className={`${bg} rounded-2xl p-4 flex items-start gap-3`}>
          <Icon size={18} className={`${color} mt-0.5 shrink-0`} />
          <div>
            <p className="text-xs text-slate-500">{label}</p>
            <p className={`text-lg font-bold ${color}`}>{value}</p>
          </div>
        </div>
      ))}
    </div>
  )
}

function PolicySummaryBar({ policies }) {
  const active = policies.filter(p => p.status === 'ACTIVE').length
  const totalPremium = policies.reduce((s, p) => s + (Number(p.annual_premium) || 0), 0)
  const totalIDV = policies.reduce((s, p) => s + (Number(p.idv) || 0), 0)
  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
      {[
        { label: 'Total Policies', value: policies.length,                                              icon: FileText,   color: 'text-blue-600',   bg: 'bg-blue-50' },
        { label: 'Active',         value: active,                                                        icon: CheckCircle, color: 'text-emerald-600', bg: 'bg-emerald-50' },
        { label: 'Total Premium',  value: totalPremium ? `IDR ${(totalPremium / 1e6).toFixed(1)}M` : '—', icon: DollarSign, color: 'text-violet-600', bg: 'bg-violet-50' },
        { label: 'Total IDV',      value: totalIDV     ? `IDR ${(totalIDV     / 1e6).toFixed(0)}M` : '—', icon: TrendingUp, color: 'text-indigo-600', bg: 'bg-indigo-50' },
      ].map(({ label, value, icon: Icon, color, bg }) => (
        <div key={label} className={`${bg} rounded-2xl p-4 flex items-start gap-3`}>
          <Icon size={18} className={`${color} mt-0.5 shrink-0`} />
          <div>
            <p className="text-xs text-slate-500">{label}</p>
            <p className={`text-lg font-bold ${color}`}>{value}</p>
          </div>
        </div>
      ))}
    </div>
  )
}

// ─── Orders tab ───────────────────────────────────────────────────────────────

function OrdersTab() {
  const [orders, setOrders] = useState([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState(null)

  useEffect(() => {
    api.get('/api/v1/orders')
      .then(data => setOrders(Array.isArray(data) ? data : (data.orders || [])))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-48 text-slate-500">Loading orders…</div>
  }

  if (orders.length === 0) {
    return (
      <div className="text-center py-16 bg-white rounded-2xl shadow-sm">
        <RefreshCw className="text-gray-300 mx-auto mb-4" size={48} />
        <p className="text-slate-400">No orders yet.</p>
      </div>
    )
  }

  return (
    <>
      <OrderSummaryBar orders={orders} />
      <div className="space-y-3">
        {orders.map((order, i) => {
          const seam = SEAM_META[order.seam_stage] || SEAM_META.va_created
          return (
            <motion.div
              key={order.transaction_id || i}
              initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.04 }}
              onClick={() => setSelected(order)}
              className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5 cursor-pointer hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between gap-3 mb-3">
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-mono text-slate-500 truncate">{order.transaction_id}</p>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    <span className={`inline-flex items-center text-xs font-semibold px-2.5 py-0.5 rounded-full border ${seam.bg} ${seam.border} ${seam.color}`}>
                      {seam.label}
                    </span>
                    {order.payment_amount > 0 && (
                      <span className="text-sm font-bold text-slate-700">{fmtIDR(order.payment_amount)}</span>
                    )}
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs text-slate-400">{fmtDateShort(order.created_at)}</p>
                  {order.doku_va_number && (
                    <p className="text-xs font-mono text-slate-500 mt-0.5">VA: {order.doku_va_number}</p>
                  )}
                </div>
              </div>
              <SeamStageBar stage={order.seam_stage} />
            </motion.div>
          )
        })}
      </div>
      <OrderDetailPanel order={selected} onClose={() => setSelected(null)} />
    </>
  )
}

// ─── Policies tab ─────────────────────────────────────────────────────────────

function PoliciesTab() {
  const [policies, setPolicies] = useState([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState(null)
  const navigate = useNavigate()

  useEffect(() => {
    api.get('/api/v1/policies')
      .then(data => setPolicies(Array.isArray(data) ? data : (data.policies || [])))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-48 text-slate-500">Loading policies…</div>
  }

  if (policies.length === 0) {
    return (
      <div className="text-center py-16 bg-white rounded-2xl shadow-sm">
        <Shield className="text-gray-300 mx-auto mb-4" size={48} />
        <p className="text-slate-400">No policies found.</p>
        <button onClick={() => navigate('/')} className="mt-4 px-6 py-2.5 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700">
          Browse Products
        </button>
      </div>
    )
  }

  return (
    <>
      <PolicySummaryBar policies={policies} />
      <div className="space-y-3">
        {policies.map((policy, i) => {
          const meta = STATUS_META[policy.status] || STATUS_META.ACTIVE
          const StatusIcon = meta.icon
          return (
            <motion.div
              key={policy.transaction_id || i}
              initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.04 }}
              onClick={() => setSelected(policy)}
              className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5 cursor-pointer hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="font-bold text-slate-900 text-base">{policy.policy_number || 'Policy'}</p>
                    <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full border ${meta.bg} ${meta.border} ${meta.color}`}>
                      <StatusIcon size={10} />
                      {policy.status || 'ACTIVE'}
                    </span>
                  </div>
                  <p className="text-xs text-slate-400 font-mono mt-0.5 truncate">{policy.transaction_id}</p>
                  <div className="flex gap-4 mt-2.5 flex-wrap">
                    {policy.annual_premium > 0 && (
                      <div>
                        <p className="text-xs text-slate-400">Annual Premium</p>
                        <p className="text-sm font-semibold text-blue-700">IDR {Number(policy.annual_premium).toLocaleString('id-ID')}</p>
                      </div>
                    )}
                    {policy.idv > 0 && (
                      <div>
                        <p className="text-xs text-slate-400">IDV</p>
                        <p className="text-sm font-semibold text-indigo-700">IDR {Number(policy.idv).toLocaleString('id-ID')}</p>
                      </div>
                    )}
                  </div>
                </div>
                {policy.certificate_url && (
                  <a
                    href={policy.certificate_url} target="_blank" rel="noreferrer"
                    onClick={e => e.stopPropagation()}
                    className="p-2 bg-blue-50 hover:bg-blue-100 rounded-xl text-blue-600 transition-colors shrink-0"
                    title="Download certificate"
                  >
                    <Download size={16} />
                  </a>
                )}
              </div>
              <CoveragePill start={policy.coverage_start} end={policy.coverage_end} />
            </motion.div>
          )
        })}
      </div>
      <PolicyDetailPanel policy={selected} onClose={() => setSelected(null)} />
    </>
  )
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function PolicyHistoryPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [tab, setTab] = useState('orders')

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <header className="bg-white shadow-sm">
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center gap-3">
          <button onClick={() => navigate('/')} className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-2">
            <Shield className="text-blue-600" size={22} />
            <span className="font-bold text-slate-900">MotorInsure</span>
          </div>
          <span className="font-medium text-slate-700 ml-2">Order History</span>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold text-slate-900">Order History</h1>
          <div className="flex bg-white border border-slate-200 rounded-xl p-1 gap-1">
            {[
              { key: 'orders',   label: 'All Orders',  icon: RefreshCw },
              { key: 'policies', label: 'Policies',    icon: Shield },
            ].map(({ key, label, icon: Icon }) => (
              <button
                key={key}
                onClick={() => setTab(key)}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  tab === key ? 'bg-blue-600 text-white' : 'text-slate-600 hover:bg-slate-50'
                }`}
              >
                <Icon size={14} />
                {label}
              </button>
            ))}
          </div>
        </div>

        <AnimatePresence mode="wait">
          <motion.div
            key={tab}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.15 }}
          >
            {tab === 'orders' ? <OrdersTab /> : <PoliciesTab />}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  )
}
