import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Search, Shield, Globe } from 'lucide-react'
import { motion } from 'framer-motion'
import { discover } from '../api/client.js'
import ProductCard from '../components/ProductCard.jsx'

export default function HeroPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [lang, setLang] = useState(localStorage.getItem('bap_lang') || 'id')
  const [query, setQuery] = useState('')
  const [products, setProducts] = useState([])
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)
  const [vehicleFilter, setVehicleFilter] = useState('')
  const [coverageFilter, setCoverageFilter] = useState('')
  const [sortBy, setSortBy] = useState('default')

  function handleLangChange(l) {
    setLang(l)
    i18n.changeLanguage(l)
    localStorage.setItem('bap_lang', l)
  }

  // Parse Beckn v2 on_discover response: message.catalogs[].resources[]
  function parseDiscoverResponse(data) {
    const catalogs = data?.message?.catalogs || []
    return catalogs.flatMap(catalog => {
      const insurerName = catalog.provider?.descriptor?.name || catalog.descriptor?.name || catalog.id || 'Insurer'
      return (catalog.resources || []).map(resource => ({
        id: resource.id,
        descriptor: resource.descriptor,
        ...(resource.resourceAttributes || {}),
        insurer_name: insurerName,
        bpp_id: data?.context?.bpp_id || catalog.id,
      }))
    })
  }

  async function handleSearch(e) {
    e && e.preventDefault()
    setLoading(true)
    setSearched(true)
    try {
      const res = await discover(query)
      let results = parseDiscoverResponse(res.data)
      if (sortBy === 'rate_asc') results = [...results].sort((a, b) => (a.premiumRateMin ?? a.premium_rate_min ?? 0) - (b.premiumRateMin ?? b.premium_rate_min ?? 0))
      if (sortBy === 'rate_desc') results = [...results].sort((a, b) => (b.premiumRateMin ?? b.premium_rate_min ?? 0) - (a.premiumRateMin ?? a.premium_rate_min ?? 0))
      setProducts(results)
    } catch {
      setProducts([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    setLoading(true)
    discover('')
      .then(res => setProducts(parseDiscoverResponse(res.data)))
      .catch(() => setProducts([]))
      .finally(() => setLoading(false))
  }, [])

  function handleGetQuote(product) {
    navigate('/flow', { state: { selectedItem: product } })
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Shield className="text-blue-600" size={28} />
            <span className="font-bold text-xl text-slate-900">MotorInsure</span>
          </div>
          <div className="flex items-center gap-2">
            <Globe size={16} className="text-slate-400" />
            <div className="flex gap-3">
              <label className="flex items-center gap-1.5 cursor-pointer text-sm">
                <input type="radio" name="lang" value="en" checked={lang === 'en'} onChange={() => handleLangChange('en')} className="text-blue-600" />
                English
              </label>
              <label className="flex items-center gap-1.5 cursor-pointer text-sm">
                <input type="radio" name="lang" value="id" checked={lang === 'id'} onChange={() => handleLangChange('id')} className="text-blue-600" />
                Indonesia
              </label>
            </div>
          </div>
          <button onClick={() => navigate('/history')} className="text-sm text-blue-600 hover:underline font-medium">
            {t('nav.policies')}
          </button>
        </div>
      </header>

      {/* Hero section */}
      <section className="max-w-4xl mx-auto px-4 py-16 text-center">
        <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.6 }}>
          <h1 className="text-4xl md:text-5xl font-bold text-slate-900 mb-4">{t('hero.title')}</h1>
          <p className="text-xl text-slate-600 mb-10">{t('hero.subtitle')}</p>
        </motion.div>

        <motion.form initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}
          onSubmit={handleSearch} className="flex gap-3 max-w-xl mx-auto">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder={t('hero.search_placeholder')}
              className="w-full pl-10 pr-4 py-3.5 rounded-xl border border-gray-200 shadow-sm text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            />
          </div>
          <button type="submit"
            className="px-6 py-3.5 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700 transition-colors whitespace-nowrap">
            {t('hero.search')}
          </button>
        </motion.form>
      </section>

      {/* Filters */}
      <div className="max-w-6xl mx-auto px-4 mb-6 flex flex-wrap gap-3">
        <select value={vehicleFilter} onChange={e => setVehicleFilter(e.target.value)}
          className="px-4 py-2 rounded-lg border border-gray-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option value="">All Vehicle Types</option>
          <option value="TWO_WHEELER">Two Wheeler</option>
          <option value="FOUR_WHEELER">Four Wheeler</option>
          <option value="COMMERCIAL_VEHICLE">Commercial Vehicle</option>
        </select>
        <select value={coverageFilter} onChange={e => setCoverageFilter(e.target.value)}
          className="px-4 py-2 rounded-lg border border-gray-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option value="">All Coverage Types</option>
          <option value="MOTOR_COMPREHENSIVE">Comprehensive</option>
          <option value="MOTOR_THIRD_PARTY">Third Party</option>
          <option value="MOTOR_FIRE_THEFT">Fire &amp; Theft</option>
        </select>
        <select value={sortBy} onChange={e => setSortBy(e.target.value)}
          className="px-4 py-2 rounded-lg border border-gray-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option value="default">Sort: Default</option>
          <option value="rate_asc">Rate: Low to High</option>
          <option value="rate_desc">Rate: High to Low</option>
        </select>
        {(vehicleFilter || coverageFilter || sortBy !== 'default') && (
          <button onClick={() => { setVehicleFilter(''); setCoverageFilter(''); setSortBy('default') }}
            className="px-4 py-2 rounded-lg border border-red-200 bg-red-50 text-red-600 text-sm hover:bg-red-100">
            Clear Filters
          </button>
        )}
      </div>

      {/* Products grid */}
      <div className="max-w-6xl mx-auto px-4 pb-16">
        {loading ? (
          <div className="flex items-center justify-center h-48 text-slate-500">{t('common.loading')}</div>
        ) : products.length === 0 ? (
          <div className="text-center py-16 text-slate-400">
            {searched ? 'No products found. Try a different search.' : 'Search for motor insurance products above.'}
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {products.map((product, i) => (
              <motion.div key={i} initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}>
                <ProductCard product={product} onGetQuote={handleGetQuote} />
              </motion.div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
