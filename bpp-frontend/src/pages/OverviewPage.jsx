import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  FileText, TrendingUp, DollarSign, AlertCircle,
  Package, Tag, Headphones, Star, Clock, CheckCircle, RefreshCw, Banknote
} from 'lucide-react'
import { motion } from 'framer-motion'
import client from '../api/client.js'

function StatCard({ label, value, icon: Icon, color, delay = 0 }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, delay }}
      className="bg-white rounded-2xl p-6 shadow-sm border border-slate-100"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <p className="text-sm text-slate-500 font-medium truncate">{label}</p>
          <p className="text-2xl font-bold text-slate-900 mt-1 truncate">{value ?? '—'}</p>
        </div>
        <div className={`p-3 rounded-xl ${color} flex-shrink-0 ml-3`}>
          <Icon size={20} className="text-white" />
        </div>
      </div>
    </motion.div>
  )
}

function formatIDR(n) {
  if (n == null) return '—'
  return 'IDR ' + Number(n).toLocaleString('id-ID')
}

export default function OverviewPage() {
  const { t } = useTranslation()
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    client.get('/v1/dashboard/stats')
      .then(res => setStats(res.data))
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  const statCards = [
    {
      key: 'active_policies',
      label: t('dashboard.active_policies'),
      value: stats?.active_policies,
      icon: TrendingUp,
      color: 'bg-green-500',
    },
    {
      key: 'policies_issued',
      label: t('dashboard.policies_issued'),
      value: stats?.policies_issued,
      icon: FileText,
      color: 'bg-blue-500',
    },
    {
      key: 'total_premium_idr',
      label: t('dashboard.total_premium_idr'),
      value: formatIDR(stats?.total_premium_idr),
      icon: DollarSign,
      color: 'bg-emerald-500',
    },
    {
      key: 'pending_claims',
      label: t('dashboard.pending_claims'),
      value: stats?.pending_claims,
      icon: AlertCircle,
      color: 'bg-orange-500',
    },
    {
      key: 'active_resources',
      label: t('dashboard.active_resources'),
      value: stats?.active_resources,
      icon: Package,
      color: 'bg-violet-500',
    },
    {
      key: 'active_offers',
      label: t('dashboard.active_offers'),
      value: stats?.active_offers,
      icon: Tag,
      color: 'bg-cyan-500',
    },
    {
      key: 'support_tickets',
      label: t('dashboard.support_tickets'),
      value: stats?.support_tickets,
      icon: Headphones,
      color: 'bg-rose-500',
    },
    {
      key: 'avg_rating',
      label: t('dashboard.avg_rating'),
      value: stats?.avg_rating != null ? Number(stats.avg_rating).toFixed(1) + ' / 5' : '—',
      icon: Star,
      color: 'bg-amber-500',
    },
  ]

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 text-sm">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t('dashboard.title')}</h1>
        <p className="text-slate-500 text-sm mt-1">BPP Insurance Admin Dashboard</p>
      </div>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">
          {error}
        </div>
      )}

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((card, i) => (
          <StatCard key={card.key} {...card} delay={i * 0.05} />
        ))}
      </div>

      {/* SEAM Pipeline section */}
      <div className="mt-8">
        <div className="mb-4">
          <h2 className="text-lg font-bold text-slate-900">SEAM Pipeline</h2>
          <p className="text-slate-500 text-xs mt-0.5">Settlement Assurance & Escrow Mechanism — live counters</p>
        </div>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            { label: 'Awaiting Payment', value: stats?.seam_payment_pending   ?? '—', icon: Clock,        color: 'bg-amber-400' },
            { label: 'Payment Received', value: stats?.seam_payment_received  ?? '—', icon: CheckCircle,  color: 'bg-blue-500' },
            { label: 'Reconcile Pending', value: stats?.seam_reconcile_pending ?? '—', icon: RefreshCw,   color: 'bg-violet-500' },
            { label: 'Fully Settled',    value: stats?.seam_reconcile_settled ?? '—', icon: Banknote,     color: 'bg-emerald-500' },
          ].map((card, i) => (
            <StatCard key={card.label} {...card} delay={0.4 + i * 0.05} />
          ))}
        </div>
      </div>
    </div>
  )
}
