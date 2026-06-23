import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Shield, ArrowLeft, X, Download, Calendar, DollarSign, FileText, TrendingUp, CheckCircle, Clock, AlertCircle } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../api.js'
import { fmtDate } from '../utils/date.js'
import StatusBadge from '../components/StatusBadge.jsx'

const STATUS_META = {
  ACTIVE:            { icon: CheckCircle, color: 'text-emerald-500', bg: 'bg-emerald-50', border: 'border-emerald-200' },
  PENDING_ISSUANCE:  { icon: Clock,        color: 'text-amber-500',   bg: 'bg-amber-50',   border: 'border-amber-200' },
  CANCELLED:         { icon: X,            color: 'text-red-500',     bg: 'bg-red-50',     border: 'border-red-200' },
  EXPIRED:           { icon: AlertCircle,  color: 'text-slate-400',   bg: 'bg-slate-50',   border: 'border-slate-200' },
  TOTAL_LOSS:        { icon: AlertCircle,  color: 'text-red-600',     bg: 'bg-red-50',     border: 'border-red-200' },
}

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

function PolicyDetailPanel({ policy, onClose }) {
  if (!policy) return null

  const meta = STATUS_META[policy.status] || STATUS_META.ACTIVE
  const StatusIcon = meta.icon

  return (
    <AnimatePresence>
      <motion.div
        key="overlay"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/40 z-40"
        onClick={onClose}
      />
      <motion.div
        key="panel"
        initial={{ x: '100%' }}
        animate={{ x: 0 }}
        exit={{ x: '100%' }}
        transition={{ type: 'spring', damping: 25 }}
        className="fixed right-0 top-0 h-full w-full sm:w-[420px] bg-white shadow-2xl z-50 overflow-y-auto"
      >
        {/* Header */}
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
          {/* Status */}
          <div className={`flex items-center gap-3 p-3 rounded-xl border ${meta.bg} ${meta.border}`}>
            <StatusIcon size={18} className={meta.color} />
            <div>
              <p className="text-xs text-slate-500">Policy Status</p>
              <p className={`font-semibold text-sm ${meta.color}`}>{policy.status || 'ACTIVE'}</p>
            </div>
          </div>

          {/* Financial summary */}
          <div className="grid grid-cols-2 gap-3">
            <div className="bg-blue-50 rounded-xl p-4">
              <p className="text-xs text-blue-500 font-medium mb-1 flex items-center gap-1">
                <DollarSign size={10} /> Annual Premium
              </p>
              <p className="text-lg font-bold text-blue-700">
                {policy.annual_premium
                  ? `IDR ${Number(policy.annual_premium).toLocaleString('id-ID')}`
                  : '—'}
              </p>
            </div>
            <div className="bg-indigo-50 rounded-xl p-4">
              <p className="text-xs text-indigo-500 font-medium mb-1 flex items-center gap-1">
                <TrendingUp size={10} /> Insured Value (IDV)
              </p>
              <p className="text-lg font-bold text-indigo-700">
                {policy.idv
                  ? `IDR ${Number(policy.idv).toLocaleString('id-ID')}`
                  : '—'}
              </p>
            </div>
          </div>

          {/* Coverage period */}
          <div className="bg-slate-50 rounded-xl p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3 flex items-center gap-1.5">
              <Calendar size={11} /> Coverage Period
            </p>
            <div className="flex gap-4">
              <div>
                <p className="text-xs text-slate-400">Start Date</p>
                <p className="text-sm font-semibold text-slate-800">{fmtDate(policy.coverage_start)}</p>
              </div>
              <div className="w-px bg-slate-200" />
              <div>
                <p className="text-xs text-slate-400">End Date</p>
                <p className="text-sm font-semibold text-slate-800">{fmtDate(policy.coverage_end)}</p>
              </div>
            </div>
            <CoveragePill start={policy.coverage_start} end={policy.coverage_end} />
          </div>

          {/* Transaction ID */}
          <div>
            <p className="text-xs text-slate-500 mb-1">Transaction ID</p>
            <p className="text-xs font-mono bg-slate-50 border border-slate-100 rounded-lg px-3 py-2 text-slate-600 break-all">
              {policy.transaction_id}
            </p>
          </div>

          {/* Certificate */}
          {policy.certificate_url && (
            <a
              href={policy.certificate_url}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-3 p-4 border-2 border-dashed border-blue-200 rounded-xl hover:bg-blue-50 transition-colors group"
            >
              <Download size={18} className="text-blue-500 group-hover:text-blue-700" />
              <div>
                <p className="text-sm font-semibold text-blue-700">Download Certificate</p>
                <p className="text-xs text-slate-400">Policy certificate (PDF)</p>
              </div>
            </a>
          )}

          {/* Issued at */}
          <div>
            <p className="text-xs text-slate-400">Issued on</p>
            <p className="text-sm text-slate-600">{fmtDate(policy.received_at)}</p>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  )
}

function SummaryBar({ policies }) {
  const active = policies.filter(p => p.status === 'ACTIVE').length
  const totalPremium = policies.reduce((s, p) => s + (Number(p.annual_premium) || 0), 0)
  const totalIDV = policies.reduce((s, p) => s + (Number(p.idv) || 0), 0)

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
      {[
        { label: 'Total Policies', value: policies.length, icon: FileText, color: 'text-blue-600', bg: 'bg-blue-50' },
        { label: 'Active', value: active, icon: CheckCircle, color: 'text-emerald-600', bg: 'bg-emerald-50' },
        { label: 'Total Premium / yr', value: totalPremium ? `IDR ${(totalPremium / 1e6).toFixed(1)}M` : '—', icon: DollarSign, color: 'text-violet-600', bg: 'bg-violet-50' },
        { label: 'Total IDV', value: totalIDV ? `IDR ${(totalIDV / 1e6).toFixed(0)}M` : '—', icon: TrendingUp, color: 'text-indigo-600', bg: 'bg-indigo-50' },
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

export default function PolicyHistoryPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [policies, setPolicies] = useState([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState(null)

  useEffect(() => {
    api.get('/api/v1/policies')
      .then(data => setPolicies(Array.isArray(data) ? data : (data.policies || [])))
      .finally(() => setLoading(false))
  }, [])

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
          <span className="font-medium text-slate-700 ml-2">{t('policy.history')}</span>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold text-slate-900">{t('policy.history')}</h1>
          {policies.length > 0 && (
            <span className="text-sm text-slate-500 bg-white border border-slate-200 rounded-xl px-3 py-1.5">
              {policies.length} {policies.length === 1 ? 'policy' : 'policies'}
            </span>
          )}
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-48 text-slate-500">{t('common.loading')}</div>
        ) : policies.length === 0 ? (
          <div className="text-center py-16 bg-white rounded-2xl shadow-sm">
            <Shield className="text-gray-300 mx-auto mb-4" size={48} />
            <p className="text-slate-400">No policies found.</p>
            <button onClick={() => navigate('/')} className="mt-4 px-6 py-2.5 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700">
              Browse Products
            </button>
          </div>
        ) : (
          <>
            <SummaryBar policies={policies} />
            <div className="space-y-3">
              {policies.map((policy, i) => {
                const meta = STATUS_META[policy.status] || STATUS_META.ACTIVE
                const StatusIcon = meta.icon
                return (
                  <motion.div
                    key={policy.transaction_id || i}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.04 }}
                    onClick={() => setSelected(policy)}
                    className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5 cursor-pointer hover:shadow-md transition-shadow"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <p className="font-bold text-slate-900 text-base">
                            {policy.policy_number || 'Policy'}
                          </p>
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
                              <p className="text-sm font-semibold text-blue-700">
                                IDR {Number(policy.annual_premium).toLocaleString('id-ID')}
                              </p>
                            </div>
                          )}
                          {policy.idv > 0 && (
                            <div>
                              <p className="text-xs text-slate-400">IDV</p>
                              <p className="text-sm font-semibold text-indigo-700">
                                IDR {Number(policy.idv).toLocaleString('id-ID')}
                              </p>
                            </div>
                          )}
                        </div>
                      </div>

                      {policy.certificate_url && (
                        <a
                          href={policy.certificate_url}
                          target="_blank"
                          rel="noreferrer"
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
          </>
        )}
      </div>

      <PolicyDetailPanel policy={selected} onClose={() => setSelected(null)} />
    </div>
  )
}
