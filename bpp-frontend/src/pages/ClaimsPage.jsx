import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import client from '../api/client.js'
import StatusBadge from '../components/StatusBadge.jsx'

function formatIDR(n) {
  if (n == null) return '—'
  return 'IDR ' + Number(n).toLocaleString('id-ID')
}

export default function ClaimsPage() {
  const { t } = useTranslation()
  const [claims, setClaims] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    client.get('/v1/claims')
      .then(res => {
        const data = res.data
        setClaims(Array.isArray(data) ? data : (data?.claims || data?.items || []))
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 text-sm">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-900 mb-6">{t('claims.title')}</h1>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">{error}</div>
      )}

      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="bg-white rounded-2xl shadow-sm border border-slate-100 overflow-hidden"
      >
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-slate-50 border-b border-slate-100">
                {[
                  t('claims.claim_id'),
                  t('claims.policy_id'),
                  t('claims.type'),
                  t('claims.status'),
                  t('claims.amount'),
                  t('claims.submitted_at'),
                ].map(h => (
                  <th key={h} className="text-left px-5 py-3.5 text-xs font-semibold text-slate-500 uppercase tracking-wide">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {claims.map((c, i) => (
                <tr key={c.id || c.claim_id || i} className="hover:bg-slate-50 transition-colors">
                  <td className="px-5 py-3.5 font-mono text-xs text-slate-700">{c.id || c.claim_id || '—'}</td>
                  <td className="px-5 py-3.5 font-mono text-xs text-slate-600">{c.policy_id || c.policyId || '—'}</td>
                  <td className="px-5 py-3.5 text-slate-700">{c.claim_type || c.type || '—'}</td>
                  <td className="px-5 py-3.5"><StatusBadge status={c.status} /></td>
                  <td className="px-5 py-3.5 text-slate-700">{formatIDR(c.amount || c.claim_amount)}</td>
                  <td className="px-5 py-3.5 text-slate-500">
                    {c.submitted_at || c.created_at
                      ? new Date(c.submitted_at || c.created_at).toLocaleDateString('id-ID')
                      : '—'}
                  </td>
                </tr>
              ))}
              {claims.length === 0 && (
                <tr>
                  <td colSpan={6} className="py-16 text-center text-slate-400 text-sm">
                    {t('common.no_data')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </motion.div>
    </div>
  )
}
