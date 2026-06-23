import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import { Search } from 'lucide-react'
import client from '../api/client.js'
import StatusBadge from '../components/StatusBadge.jsx'
import { fmtDate, fmtDateTime } from '../utils/date.js'

function formatIDR(n) {
  if (n == null) return '—'
  return 'IDR ' + Number(n).toLocaleString('id-ID')
}

export default function PoliciesPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [policies, setPolicies] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')

  useEffect(() => {
    client.get('/v1/policies')
      .then(res => {
        const data = res.data
        setPolicies(Array.isArray(data) ? data : (data?.policies || data?.items || []))
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  const filtered = policies.filter(p => {
    const q = search.toLowerCase()
    return (
      (p.policy_number || '').toLowerCase().includes(q) ||
      (p.policyholder_nik || p.nik || '').toLowerCase().includes(q) ||
      (p.vehicle_vin || p.vin || '').toLowerCase().includes(q)
    )
  })

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 text-sm">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t('policies.title')}</h1>
        <div className="relative">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search policies..."
            className="pl-9 pr-4 py-2 text-sm border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
          />
        </div>
      </div>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">
          {error}
        </div>
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
                  t('policies.policy_number'),
                  t('policies.status'),
                  t('policies.policyholder_nik'),
                  t('policies.vehicle_vin'),
                  t('policies.idv'),
                  t('policies.created_at'),
                ].map(h => (
                  <th key={h} className="text-left px-5 py-3.5 text-xs font-semibold text-slate-500 uppercase tracking-wide">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {filtered.map((p, i) => (
                <tr
                  key={p.id || p.policy_id || i}
                  onClick={() => navigate(`/policies/${p.id || p.policy_id || p.policy_number}`)}
                  className="hover:bg-blue-50 cursor-pointer transition-colors"
                >
                  <td className="px-5 py-3.5 font-mono text-xs font-semibold text-slate-800">
                    {p.policy_number || '—'}
                  </td>
                  <td className="px-5 py-3.5">
                    <StatusBadge status={p.status} />
                  </td>
                  <td className="px-5 py-3.5 font-mono text-xs text-slate-600">
                    {p.policyholder_nik || p.nik || '—'}
                  </td>
                  <td className="px-5 py-3.5 font-mono text-xs text-slate-600">
                    {p.vehicle_vin || p.vin || '—'}
                  </td>
                  <td className="px-5 py-3.5 text-slate-700">
                    {formatIDR(p.idv)}
                  </td>
                  <td className="px-5 py-3.5 text-slate-500">
                    {fmtDate(p.created_at)}
                  </td>
                </tr>
              ))}
              {filtered.length === 0 && (
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
