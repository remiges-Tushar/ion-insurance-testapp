import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import { Star, FileText, TrendingUp, MessageSquare } from 'lucide-react'
import client from '../api/client.js'

function StarDisplay({ score }) {
  const n = Number(score) || 0
  return (
    <div className="flex items-center gap-0.5">
      {[1, 2, 3, 4, 5].map(s => (
        <Star
          key={s}
          size={14}
          className={s <= n ? 'text-amber-400 fill-amber-400' : 'text-slate-200 fill-slate-200'}
        />
      ))}
      <span className="text-xs text-slate-500 ml-1.5 font-semibold">{n}/5</span>
    </div>
  )
}

function ScoreDistribution({ ratings }) {
  const counts = [5, 4, 3, 2, 1].map(star => ({
    star,
    count: ratings.filter(r => Number(r.score) === star).length,
  }))
  const max = Math.max(...counts.map(c => c.count), 1)
  return (
    <div className="space-y-1.5">
      {counts.map(({ star, count }) => (
        <div key={star} className="flex items-center gap-2 text-xs">
          <span className="w-4 text-right text-slate-500">{star}</span>
          <Star size={10} className="text-amber-400 fill-amber-400 shrink-0" />
          <div className="flex-1 h-2 bg-slate-100 rounded-full overflow-hidden">
            <div
              className="h-full bg-amber-400 rounded-full transition-all"
              style={{ width: `${(count / max) * 100}%` }}
            />
          </div>
          <span className="w-5 text-slate-400">{count}</span>
        </div>
      ))}
    </div>
  )
}

export default function RatingsPage() {
  const { t } = useTranslation()
  const [ratings, setRatings] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    client.get('/v1/ratings')
      .then(res => {
        const data = res.data
        setRatings(Array.isArray(data) ? data : (data?.ratings || data?.items || []))
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  const ratedCount = ratings.filter(r => Number(r.score) > 0).length
  const avgScore = ratedCount > 0
    ? (ratings.reduce((sum, r) => sum + (Number(r.score) || 0), 0) / ratedCount).toFixed(1)
    : null

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
        <h1 className="text-2xl font-bold text-slate-900">{t('ratings.title')}</h1>
      </div>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">{error}</div>
      )}

      {/* Summary cards */}
      {ratings.length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
          {/* Average score */}
          <div className="bg-amber-50 border border-amber-100 rounded-2xl p-5 flex items-center gap-4">
            <div className="w-12 h-12 bg-amber-100 rounded-xl flex items-center justify-center">
              <Star size={22} className="text-amber-500 fill-amber-500" />
            </div>
            <div>
              <p className="text-xs text-amber-600 font-medium">Average Score</p>
              <p className="text-3xl font-bold text-amber-700">{avgScore ?? '—'}</p>
              <p className="text-xs text-amber-500">{ratedCount} rated review{ratedCount !== 1 ? 's' : ''}</p>
            </div>
          </div>

          {/* Total reviews */}
          <div className="bg-slate-50 border border-slate-100 rounded-2xl p-5 flex items-center gap-4">
            <div className="w-12 h-12 bg-slate-100 rounded-xl flex items-center justify-center">
              <MessageSquare size={20} className="text-slate-500" />
            </div>
            <div>
              <p className="text-xs text-slate-500 font-medium">Total Reviews</p>
              <p className="text-3xl font-bold text-slate-700">{ratings.length}</p>
              <p className="text-xs text-slate-400">
                {ratings.filter(r => r.feedback).length} with comments
              </p>
            </div>
          </div>

          {/* Distribution */}
          <div className="bg-white border border-slate-100 rounded-2xl p-5">
            <p className="text-xs text-slate-500 font-medium mb-3">Score Distribution</p>
            <ScoreDistribution ratings={ratings} />
          </div>
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
                {['Policy', 'Score', 'Comment', 'Submitted'].map(h => (
                  <th key={h} className="text-left px-5 py-3.5 text-xs font-semibold text-slate-500 uppercase tracking-wide">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {ratings.map((r, i) => (
                <tr key={r.id || i} className="hover:bg-slate-50 transition-colors">
                  {/* Policy column — the most important context */}
                  <td className="px-5 py-4">
                    {r.policy_number ? (
                      <div>
                        <div className="flex items-center gap-1.5">
                          <FileText size={13} className="text-blue-500 shrink-0" />
                          <span className="font-semibold text-slate-800 text-sm">{r.policy_number}</span>
                        </div>
                        {r.transaction_id && (
                          <p className="text-xs font-mono text-slate-400 mt-0.5 truncate max-w-[180px]">
                            {r.transaction_id}
                          </p>
                        )}
                      </div>
                    ) : (
                      <span className="text-slate-400 text-xs italic">No policy linked</span>
                    )}
                  </td>
                  <td className="px-5 py-4"><StarDisplay score={r.score} /></td>
                  <td className="px-5 py-4 text-slate-600 max-w-xs">
                    {r.feedback
                      ? <p className="line-clamp-2 text-sm">{r.feedback}</p>
                      : <span className="text-slate-300 italic text-xs">No comment</span>
                    }
                  </td>
                  <td className="px-5 py-4 text-slate-400 text-xs whitespace-nowrap">
                    {r.created_at ? new Date(r.created_at).toLocaleDateString('id-ID', {
                      day: '2-digit', month: 'short', year: 'numeric',
                    }) : '—'}
                  </td>
                </tr>
              ))}
              {ratings.length === 0 && (
                <tr>
                  <td colSpan={4} className="py-16 text-center text-slate-400 text-sm">
                    {t('common.no_data')}
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
