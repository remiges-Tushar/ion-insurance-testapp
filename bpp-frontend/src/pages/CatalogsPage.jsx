import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api.js'
import StatusBadge from '../components/StatusBadge.jsx'

export default function CatalogsPage() {
  const { t } = useTranslation()
  const [catalogs, setCatalogs] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get('/api/v1/catalogs')
      .then(data => setCatalogs(Array.isArray(data) ? data : (data.catalogs || [])))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="flex items-center justify-center h-64 text-slate-500">{t('common.loading')}</div>

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-900 mb-6">{t('nav.catalogs')}</h1>
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              {['ID', 'Name', 'Version', 'CDS Status', 'Published At', 'Created'].map(h => (
                <th key={h} className="text-left px-4 py-3 text-slate-500 font-medium">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {catalogs.map((c, i) => (
              <tr key={i} className="border-b border-gray-50 hover:bg-gray-50">
                <td className="px-4 py-3 font-mono text-xs">{c.id || c.catalog_id || '—'}</td>
                <td className="px-4 py-3">{c.name || '—'}</td>
                <td className="px-4 py-3">{c.version || '—'}</td>
                <td className="px-4 py-3"><StatusBadge status={c.cds_status || c.status} /></td>
                <td className="px-4 py-3 text-slate-500">{c.published_at ? new Date(c.published_at).toLocaleString() : '—'}</td>
                <td className="px-4 py-3 text-slate-500">{c.created_at ? new Date(c.created_at).toLocaleDateString() : '—'}</td>
              </tr>
            ))}
            {catalogs.length === 0 && <tr><td colSpan={6} className="py-12 text-center text-slate-400">No catalogs found</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  )
}
