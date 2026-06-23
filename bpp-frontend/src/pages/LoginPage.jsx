import React, { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Shield, Eye, EyeOff, Wand2, AlertCircle } from 'lucide-react'
import { motion } from 'framer-motion'
import client from '../api/client.js'

const DEMO_CREDS = { email: 'admin@asuransimaju.id', password: 'password123' }

export default function LoginPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [lang, setLang] = useState(localStorage.getItem('bpp_lang') || 'id')

  function handleLangChange(l) {
    setLang(l)
    i18n.changeLanguage(l)
    localStorage.setItem('bpp_lang', l)
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await client.post('/v1/auth/login', { email, password })
      const token = res.data?.token || res.data?.access_token
      if (token) {
        localStorage.setItem('bpp_token', token)
        localStorage.setItem('bpp_email', email)
        navigate('/dashboard')
      } else {
        setError(res.data?.detail || res.data?.message || t('common.error'))
      }
    } catch (err) {
      setError(err.response?.data?.detail || err.response?.data?.message || t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-blue-900 flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
        className="bg-white rounded-2xl shadow-2xl w-full max-w-md p-8"
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-blue-600 rounded-xl flex items-center justify-center flex-shrink-0">
              <Shield size={24} className="text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-slate-900">BPP Insurance</h1>
              <p className="text-slate-500 text-sm">{t('auth.sign_in_subtitle')}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => { setEmail(DEMO_CREDS.email); setPassword(DEMO_CREDS.password) }}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-violet-700 bg-violet-50 border border-violet-200 rounded-lg hover:bg-violet-100 transition"
          >
            <Wand2 size={13} />
            Auto Fill
          </button>
        </div>

        {/* Language selector */}
        <div className="mb-6 p-3 bg-slate-50 rounded-xl">
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">{t('auth.language')}</p>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                name="lang"
                value="en"
                checked={lang === 'en'}
                onChange={() => handleLangChange('en')}
                className="accent-blue-600"
              />
              <span className="text-sm text-slate-700">{t('auth.english')}</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                name="lang"
                value="id"
                checked={lang === 'id'}
                onChange={() => handleLangChange('id')}
                className="accent-blue-600"
              />
              <span className="text-sm text-slate-700">{t('auth.indonesia')}</span>
            </label>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">{t('auth.email')}</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
              autoComplete="email"
              className="w-full border border-slate-300 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition"
              placeholder="admin@insurer.co.id"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">{t('auth.password')}</label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={e => setPassword(e.target.value)}
                required
                autoComplete="current-password"
                className="w-full border border-slate-300 rounded-xl px-4 py-2.5 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition"
              />
              <button
                type="button"
                onClick={() => setShowPassword(v => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
              >
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>

          {error && (
            <div className="flex items-start gap-2 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">
              <AlertCircle size={16} className="shrink-0 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2.5 rounded-xl font-semibold hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-sm mt-2"
          >
            {loading ? t('common.loading') : t('auth.login')}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-slate-600">
          {t('auth.no_account')}{' '}
          <Link to="/register" className="text-blue-600 hover:underline font-semibold">
            {t('auth.register')}
          </Link>
        </p>
      </motion.div>
    </div>
  )
}
