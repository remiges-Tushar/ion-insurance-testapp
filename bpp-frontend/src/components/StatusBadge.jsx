import React from 'react'
import { useTranslation } from 'react-i18next'

const STATUS_STYLES = {
  ACTIVE:            'bg-green-100 text-green-700',
  PUBLISHED:         'bg-green-100 text-green-700',
  ACCEPTED:          'bg-blue-100 text-blue-700',
  PENDING:           'bg-yellow-100 text-yellow-700',
  PENDING_ISSUANCE:  'bg-yellow-100 text-yellow-700',
  OPEN:              'bg-yellow-100 text-yellow-700',
  EXPIRED:           'bg-slate-100 text-slate-600',
  DRAFT:             'bg-slate-100 text-slate-600',
  CLOSED:            'bg-slate-100 text-slate-600',
  RESOLVED:          'bg-teal-100 text-teal-700',
  CANCELLED:         'bg-red-100 text-red-700',
  REJECTED:          'bg-red-100 text-red-700',
  TOTAL_LOSS:        'bg-orange-100 text-orange-700',
}

export default function StatusBadge({ status }) {
  const { t } = useTranslation()
  if (!status) return <span className="text-slate-400 text-xs">—</span>
  const style = STATUS_STYLES[status] || 'bg-slate-100 text-slate-600'
  const label = t(`status.${status}`, { defaultValue: status.replace(/_/g, ' ') })
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${style}`}>
      {label}
    </span>
  )
}
