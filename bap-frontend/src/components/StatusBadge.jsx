import React from 'react'
import { useTranslation } from 'react-i18next'

const statusColors = {
  CONFIRMED: 'bg-green-100 text-green-800',
  ACTIVE: 'bg-green-100 text-green-800',
  PENDING: 'bg-yellow-100 text-yellow-800',
  PENDING_ISSUANCE: 'bg-yellow-100 text-yellow-800',
  CANCELLED: 'bg-red-100 text-red-800',
  EXPIRED: 'bg-gray-100 text-gray-800',
}

export default function StatusBadge({ status }) {
  const { t } = useTranslation()
  const color = statusColors[status] || 'bg-gray-100 text-gray-800'
  const label = t(`status.${status}`, { defaultValue: status })
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${color}`}>
      {label}
    </span>
  )
}
