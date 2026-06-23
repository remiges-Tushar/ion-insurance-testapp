import React from 'react'
import { Routes, Route, Link, useLocation, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Shield } from 'lucide-react'
import HeroPage from './pages/HeroPage.jsx'
import PolicyFlowPage from './pages/PolicyFlowPage.jsx'
import PolicyHistoryPage from './pages/PolicyHistoryPage.jsx'

function NavBar() {
  const { t } = useTranslation()
  const location = useLocation()

  return (
    <nav className="bg-white shadow-sm border-b border-gray-100 sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <Link to="/" className="flex items-center gap-2 text-blue-600 hover:text-blue-700 transition-colors">
            <Shield className="w-7 h-7" />
            <span className="font-bold text-lg tracking-tight">InsureMarket</span>
          </Link>
          <div className="flex items-center gap-6">
            <Link
              to="/"
              className={`text-sm font-medium transition-colors ${
                location.pathname === '/'
                  ? 'text-blue-600 border-b-2 border-blue-600 pb-0.5'
                  : 'text-gray-600 hover:text-blue-600'
              }`}
            >
              {t('nav.marketplace')}
            </Link>
            <Link
              to="/history"
              className={`text-sm font-medium transition-colors ${
                location.pathname === '/history'
                  ? 'text-blue-600 border-b-2 border-blue-600 pb-0.5'
                  : 'text-gray-600 hover:text-blue-600'
              }`}
            >
              {t('nav.policies')}
            </Link>
          </div>
        </div>
      </div>
    </nav>
  )
}

export default function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <NavBar />
      <Routes>
        <Route path="/" element={<HeroPage />} />
        <Route path="/policy/:txnId" element={<PolicyFlowPage />} />
        <Route path="/flow" element={<PolicyFlowPage />} />
        <Route path="/history" element={<PolicyHistoryPage />} />
        <Route path="/policies" element={<Navigate to="/history" replace />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </div>
  )
}
