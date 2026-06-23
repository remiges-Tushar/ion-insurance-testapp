import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { X, Shield, Car, FileText, Phone } from 'lucide-react'

function formatIDR(amount) {
  if (!amount && amount !== 0) return '—'
  const num = typeof amount === 'string' ? parseFloat(amount.replace(/[^0-9.-]/g, '')) : Number(amount)
  if (isNaN(num)) return '—'
  return 'Rp ' + num.toLocaleString('id-ID')
}

function DetailRow({ label, value }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs text-gray-400 font-medium uppercase tracking-wide">{label}</span>
      <span className="text-sm text-gray-800 font-medium">{value || '—'}</span>
    </div>
  )
}

function Section({ title, icon: Icon, children }) {
  return (
    <div className="bg-gray-50 rounded-xl p-4 space-y-3">
      <div className="flex items-center gap-2 mb-3">
        {Icon && <Icon size={16} className="text-blue-600" />}
        <h3 className="text-sm font-semibold text-gray-700">{title}</h3>
      </div>
      {children}
    </div>
  )
}

export default function PolicyHistoryDrawer({ policy, onClose }) {
  const { t } = useTranslation()

  if (!policy) return null

  const vehicle = policy.vehicle || policy.vehicle_details || {}
  const coverage = policy.coverage || policy.coverage_details || {}
  const insurer = policy.insurer || policy.insurer_details || {}

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-50 flex">
        {/* Backdrop */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="flex-1 bg-black/40 backdrop-blur-sm"
          onClick={onClose}
        />

        {/* Drawer panel */}
        <motion.div
          initial={{ x: 400 }}
          animate={{ x: 0 }}
          exit={{ x: 400 }}
          transition={{ type: 'spring', stiffness: 300, damping: 30 }}
          className="w-full max-w-md bg-white shadow-2xl flex flex-col overflow-hidden"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100 bg-gradient-to-r from-blue-600 to-teal-600">
            <div className="flex items-center gap-3">
              <Shield className="w-5 h-5 text-white" />
              <h2 className="text-base font-semibold text-white">{t('drawer.title')}</h2>
            </div>
            <button
              onClick={onClose}
              className="p-1.5 rounded-lg bg-white/20 hover:bg-white/30 text-white transition-colors"
              aria-label={t('drawer.close')}
            >
              <X size={18} />
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto px-6 py-5 space-y-4">

            {/* Policy Info */}
            <Section title={t('drawer.policy_info')} icon={FileText}>
              <DetailRow label={t('policies.number')} value={policy.policy_number || policy.policyNumber || policy.id} />
              <DetailRow label={t('policies.status')} value={
                <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${
                  policy.status === 'ACTIVE' ? 'bg-green-100 text-green-700' :
                  policy.status === 'PENDING' || policy.status === 'PENDING_ISSUANCE' ? 'bg-yellow-100 text-yellow-700' :
                  policy.status === 'CANCELLED' ? 'bg-red-100 text-red-700' :
                  policy.status === 'EXPIRED' ? 'bg-gray-100 text-gray-600' :
                  'bg-blue-100 text-blue-700'
                }`}>
                  {t(`status.${policy.status}`, policy.status)}
                </span>
              } />
              <DetailRow
                label={t('policies.created')}
                value={policy.created_at || policy.createdAt
                  ? new Date(policy.created_at || policy.createdAt).toLocaleDateString()
                  : '—'}
              />
              <DetailRow label={t('policies.idv')} value={formatIDR(policy.idv || policy.sum_insured)} />
              <DetailRow label={t('drawer.premium')} value={formatIDR(policy.annual_premium || policy.premium)} />
            </Section>

            {/* Vehicle Info */}
            {(vehicle.make || vehicle.model || vehicle.year || vehicle.license_plate) && (
              <Section title={t('drawer.vehicle_info')} icon={Car}>
                <DetailRow label={t('vehicle.make')} value={vehicle.make} />
                <DetailRow label={t('vehicle.model')} value={vehicle.model} />
                <DetailRow label={t('vehicle.year')} value={vehicle.year} />
                <DetailRow label={t('vehicle.plate')} value={vehicle.license_plate || vehicle.licensePlate} />
                <DetailRow label={t('vehicle.stnk')} value={vehicle.stnk_number || vehicle.stnkNumber} />
              </Section>
            )}

            {/* Coverage Details */}
            <Section title={t('drawer.coverage')} icon={Shield}>
              <DetailRow label={t('policy.coverage_start')} value={
                coverage.start || coverage.coverage_start || policy.coverage_start
                  ? new Date(coverage.start || coverage.coverage_start || policy.coverage_start).toLocaleDateString()
                  : '—'
              } />
              <DetailRow label={t('policy.coverage_end')} value={
                coverage.end || coverage.coverage_end || policy.coverage_end
                  ? new Date(coverage.end || coverage.coverage_end || policy.coverage_end).toLocaleDateString()
                  : '—'
              } />
              {(coverage.type || policy.coverage_type || policy.product_type) && (
                <DetailRow label="Coverage Type" value={coverage.type || policy.coverage_type || policy.product_type} />
              )}
            </Section>

            {/* Insurer Info */}
            {(insurer.name || insurer.contact || policy.insurer_name) && (
              <Section title={t('drawer.insurer')} icon={Phone}>
                <DetailRow label="Insurer" value={insurer.name || policy.insurer_name} />
                <DetailRow label="Contact" value={insurer.contact || policy.insurer_contact} />
                <DetailRow label="Email" value={insurer.email || policy.insurer_email} />
              </Section>
            )}

            {/* Certificate download */}
            {(policy.certificate_url || policy.certificateUrl) && (
              <a
                href={policy.certificate_url || policy.certificateUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-center gap-2 w-full bg-teal-600 hover:bg-teal-700 text-white rounded-xl py-3 text-sm font-medium transition-colors"
              >
                <FileText size={16} />
                {t('policy.certificate')}
              </a>
            )}
          </div>

          {/* Footer */}
          <div className="px-6 py-4 border-t border-gray-100">
            <button
              onClick={onClose}
              className="w-full border border-gray-300 rounded-xl py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            >
              {t('drawer.close')}
            </button>
          </div>
        </motion.div>
      </div>
    </AnimatePresence>
  )
}
