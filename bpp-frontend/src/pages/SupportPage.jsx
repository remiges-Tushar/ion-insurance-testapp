import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import { Ticket, FileText, CheckCircle, AlertCircle, Clock, MessageSquare } from 'lucide-react'
import client from '../api/client.js'

const STATUS_CONFIG = {
  OPEN:        { label: 'Open',        bg: 'bg-amber-100',   text: 'text-amber-700',   border: 'border-amber-200',   icon: Clock },
  CLOSED:      { label: 'Closed',      bg: 'bg-emerald-100', text: 'text-emerald-700', border: 'border-emerald-200', icon: CheckCircle },
  IN_PROGRESS: { label: 'In Progress', bg: 'bg-blue-100',    text: 'text-blue-700',    border: 'border-blue-200',    icon: AlertCircle },
  RESOLVED:    { label: 'Resolved',    bg: 'bg-teal-100',    text: 'text-teal-700',    border: 'border-teal-200',    icon: CheckCircle },
}

function StatusPill({ status }) {
  const cfg = STATUS_CONFIG[status?.toUpperCase()] || {
    label: status || 'Unknown',
    bg: 'bg-slate-100', text: 'text-slate-600', border: 'border-slate-200', icon: Clock,
  }
  const Icon = cfg.icon
  return (
    <span className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold border ${cfg.bg} ${cfg.text} ${cfg.border}`}>
      <Icon size={10} />
      {cfg.label}
    </span>
  )
}

function StatusBreakdown({ tickets }) {
  const statuses = ['OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED']
  const counts = statuses.map(s => ({
    s,
    count: tickets.filter(t => (t.status || 'OPEN').toUpperCase() === s).length,
  })).filter(x => x.count > 0)
  const max = Math.max(...counts.map(c => c.count), 1)

  if (counts.length === 0) return null
  return (
    <div className="space-y-1.5">
      {counts.map(({ s, count }) => {
        const cfg = STATUS_CONFIG[s] || { label: s, bg: 'bg-slate-100', text: 'text-slate-600' }
        return (
          <div key={s} className="flex items-center gap-2 text-xs">
            <span className={`w-20 text-right font-medium ${cfg.text}`}>{cfg.label}</span>
            <div className="flex-1 h-2 bg-slate-100 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${cfg.bg.replace('bg-', 'bg-').replace('-100', '-400')}`}
                style={{ width: `${(count / max) * 100}%` }}
              />
            </div>
            <span className="w-5 text-slate-400 font-medium">{count}</span>
          </div>
        )
      })}
    </div>
  )
}

export default function SupportPage() {
  const { t } = useTranslation()
  const [tickets, setTickets] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('ALL')

  useEffect(() => {
    client.get('/v1/support-tickets')
      .then(res => {
        const data = res.data
        setTickets(Array.isArray(data) ? data : (data?.tickets || data?.support_tickets || data?.items || []))
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  const open   = tickets.filter(t => (t.status || 'OPEN').toUpperCase() === 'OPEN').length
  const closed = tickets.filter(t => ['CLOSED', 'RESOLVED'].includes((t.status || '').toUpperCase())).length
  const withPolicy = tickets.filter(t => t.policy_number).length

  const displayed = filter === 'ALL' ? tickets : tickets.filter(t => (t.status || 'OPEN').toUpperCase() === filter)

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
        <h1 className="text-2xl font-bold text-slate-900">{t('support.title')}</h1>
      </div>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">{error}</div>
      )}

      {/* Summary cards */}
      {tickets.length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
          {/* Total / open */}
          <div className="bg-amber-50 border border-amber-100 rounded-2xl p-5 flex items-center gap-4">
            <div className="w-12 h-12 bg-amber-100 rounded-xl flex items-center justify-center">
              <Ticket size={22} className="text-amber-600" />
            </div>
            <div>
              <p className="text-xs text-amber-600 font-medium">Open Tickets</p>
              <p className="text-3xl font-bold text-amber-700">{open}</p>
              <p className="text-xs text-amber-500">{tickets.length} total submitted</p>
            </div>
          </div>

          {/* Resolved */}
          <div className="bg-emerald-50 border border-emerald-100 rounded-2xl p-5 flex items-center gap-4">
            <div className="w-12 h-12 bg-emerald-100 rounded-xl flex items-center justify-center">
              <CheckCircle size={22} className="text-emerald-600" />
            </div>
            <div>
              <p className="text-xs text-emerald-600 font-medium">Resolved / Closed</p>
              <p className="text-3xl font-bold text-emerald-700">{closed}</p>
              <p className="text-xs text-emerald-500">
                {tickets.length > 0 ? Math.round((closed / tickets.length) * 100) : 0}% resolution rate
              </p>
            </div>
          </div>

          {/* Status breakdown */}
          <div className="bg-white border border-slate-100 rounded-2xl p-5">
            <p className="text-xs text-slate-500 font-medium mb-3 flex items-center gap-1.5">
              <MessageSquare size={11} /> By Status
            </p>
            <StatusBreakdown tickets={tickets} />
            {withPolicy < tickets.length && (
              <p className="text-xs text-slate-400 mt-2.5 pt-2 border-t border-slate-50">
                {tickets.length - withPolicy} ticket{tickets.length - withPolicy !== 1 ? 's' : ''} without linked policy
              </p>
            )}
          </div>
        </div>
      )}

      {/* Filter tabs */}
      {tickets.length > 0 && (
        <div className="flex gap-2 mb-4 flex-wrap">
          {['ALL', 'OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED'].map(s => {
            const count = s === 'ALL' ? tickets.length : tickets.filter(t => (t.status || 'OPEN').toUpperCase() === s).length
            if (s !== 'ALL' && count === 0) return null
            const cfg = STATUS_CONFIG[s] || { label: s, bg: 'bg-slate-100', text: 'text-slate-600' }
            const active = filter === s
            return (
              <button
                key={s}
                onClick={() => setFilter(s)}
                className={`px-3 py-1.5 rounded-xl text-xs font-semibold border transition-colors ${
                  active
                    ? `${cfg.bg} ${cfg.text} ${cfg.border || 'border-transparent'}`
                    : 'bg-white text-slate-500 border-slate-200 hover:bg-slate-50'
                }`}
              >
                {s === 'ALL' ? 'All' : cfg.label} <span className="ml-1 opacity-70">{count}</span>
              </button>
            )
          })}
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
                {['Policy', 'Description', 'Status', 'Submitted'].map(h => (
                  <th key={h} className="text-left px-5 py-3.5 text-xs font-semibold text-slate-500 uppercase tracking-wide">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {displayed.map((ticket, i) => (
                <tr key={ticket.id || i} className="hover:bg-slate-50 transition-colors">
                  {/* Policy column */}
                  <td className="px-5 py-4 min-w-[180px]">
                    {ticket.policy_number ? (
                      <div>
                        <div className="flex items-center gap-1.5">
                          <FileText size={13} className="text-blue-500 shrink-0" />
                          <span className="font-semibold text-slate-800 text-sm">{ticket.policy_number}</span>
                        </div>
                        {ticket.transaction_id && (
                          <p className="text-xs font-mono text-slate-400 mt-0.5 truncate max-w-[180px]">
                            {ticket.transaction_id}
                          </p>
                        )}
                      </div>
                    ) : (
                      <div>
                        <span className="text-slate-400 text-xs italic">No policy linked</span>
                        <p className="text-xs font-mono text-slate-300 mt-0.5">#{ticket.id}</p>
                      </div>
                    )}
                  </td>

                  {/* Description */}
                  <td className="px-5 py-4 max-w-xs">
                    <p className="text-slate-800 text-sm line-clamp-2">
                      {ticket.description || '—'}
                    </p>
                  </td>

                  {/* Status */}
                  <td className="px-5 py-4 whitespace-nowrap">
                    <StatusPill status={ticket.status || 'OPEN'} />
                  </td>

                  {/* Date */}
                  <td className="px-5 py-4 text-slate-400 text-xs whitespace-nowrap">
                    {ticket.created_at ? new Date(ticket.created_at).toLocaleDateString('id-ID', {
                      day: '2-digit', month: 'short', year: 'numeric',
                    }) : '—'}
                  </td>
                </tr>
              ))}
              {displayed.length === 0 && (
                <tr>
                  <td colSpan={4} className="py-16 text-center text-slate-400 text-sm">
                    {tickets.length === 0 ? t('common.no_data') : 'No tickets match this filter.'}
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
