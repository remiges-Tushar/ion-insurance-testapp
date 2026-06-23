import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, FileText } from 'lucide-react'
import { motion } from 'framer-motion'
import client from '../api/client.js'
import StatusBadge from '../components/StatusBadge.jsx'

function DetailRow({ label, value }) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center py-3.5 border-b border-slate-100 last:border-0">
      <dt className="text-sm font-medium text-slate-500 sm:w-48 flex-shrink-0">{label}</dt>
      <dd className="text-sm text-slate-900 mt-1 sm:mt-0">{value ?? '—'}</dd>
    </div>
  )
}

function formatIDR(n) {
  if (n == null) return '—'
  return 'IDR ' + Number(n).toLocaleString('id-ID')
}

export default function PolicyDetailPage() {
  const { id } = useParams()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [policy, setPolicy] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    client.get(`/v1/policies/${id}`)
      .then(res => setPolicy(res.data))
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 text-sm">{t('common.loading')}</div>
      </div>
    )
  }

  if (error || !policy) {
    return (
      <div className="text-center py-16">
        <p className="text-slate-500 mb-4">{error || 'Policy not found'}</p>
        <button onClick={() => navigate('/policies')} className="text-blue-600 hover:underline text-sm">
          ← {t('common.back')}
        </button>
      </div>
    )
  }

  return (
    <div>
      <button
        onClick={() => navigate('/policies')}
        className="flex items-center gap-2 text-slate-500 hover:text-slate-800 text-sm mb-6 transition-colors"
      >
        <ArrowLeft size={16} />
        {t('nav.policies')}
      </button>

      <div className="flex items-center gap-3 mb-6">
        <div className="w-10 h-10 bg-blue-100 rounded-xl flex items-center justify-center">
          <FileText size={20} className="text-blue-600" />
        </div>
        <div>
          <h1 className="text-xl font-bold text-slate-900">{policy.policy_number || id}</h1>
          <StatusBadge status={policy.status} />
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        className="bg-white rounded-2xl shadow-sm border border-slate-100 p-6"
      >
        <h2 className="text-base font-semibold text-slate-900 mb-4">{t('policies.details')}</h2>
        <dl>
          <DetailRow label={t('policies.policy_number')} value={policy.policy_number} />
          <DetailRow label={t('policies.status')} value={<StatusBadge status={policy.status} />} />
          <DetailRow label={t('policies.policyholder_nik')} value={
            <span className="font-mono">{policy.policyholder_nik || policy.nik}</span>
          } />
          <DetailRow label={t('policies.vehicle_vin')} value={
            <span className="font-mono">{policy.vehicle_vin || policy.vin}</span>
          } />
          <DetailRow label={t('policies.idv')} value={formatIDR(policy.idv)} />
          <DetailRow label={t('policies.coverage_period')} value={
            policy.coverage_start && policy.coverage_end
              ? `${new Date(policy.coverage_start).toLocaleDateString('id-ID')} — ${new Date(policy.coverage_end).toLocaleDateString('id-ID')}`
              : '—'
          } />
          <DetailRow label={t('policies.transaction_id')} value={
            <span className="font-mono text-xs">{policy.transaction_id || policy.txn_id}</span>
          } />
          <DetailRow label={t('policies.created_at')} value={
            policy.created_at ? new Date(policy.created_at).toLocaleString('id-ID') : '—'
          } />
        </dl>
      </motion.div>
    </div>
  )
}
