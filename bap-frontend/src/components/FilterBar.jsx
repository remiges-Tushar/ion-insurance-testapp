import React from 'react'
import { useTranslation } from 'react-i18next'
import { SlidersHorizontal } from 'lucide-react'

export default function FilterBar({ filters, onChange }) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-wrap gap-3 items-center bg-white rounded-xl border border-gray-200 px-4 py-3 shadow-sm">
      <SlidersHorizontal className="w-4 h-4 text-gray-500 shrink-0" />

      <select
        value={filters.vehicleType || ''}
        onChange={e => onChange({ ...filters, vehicleType: e.target.value })}
        className="text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
      >
        <option value="">{t('filter.all_vehicles')}</option>
        <option value="TWO_WHEELER">{t('filter.two_wheeler')}</option>
        <option value="FOUR_WHEELER">{t('filter.four_wheeler')}</option>
        <option value="COMMERCIAL_VEHICLE">{t('filter.commercial')}</option>
      </select>

      <select
        value={filters.coverageType || ''}
        onChange={e => onChange({ ...filters, coverageType: e.target.value })}
        className="text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
      >
        <option value="">{t('filter.all_types')}</option>
        <option value="MOTOR_COMPREHENSIVE">{t('filter.comprehensive')}</option>
        <option value="MOTOR_THIRD_PARTY">{t('filter.third_party')}</option>
        <option value="MOTOR_FIRE_THEFT">{t('filter.fire_theft')}</option>
      </select>

      <select
        value={filters.sort || ''}
        onChange={e => onChange({ ...filters, sort: e.target.value })}
        className="text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
      >
        <option value="">{t('filter.sort_default')}</option>
        <option value="rate_asc">{t('filter.rate_low_high')}</option>
        <option value="rate_desc">{t('filter.rate_high_low')}</option>
      </select>
    </div>
  )
}
