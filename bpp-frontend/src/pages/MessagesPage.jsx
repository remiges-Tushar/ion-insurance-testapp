import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import { MessageSquare, ChevronDown, ChevronUp } from 'lucide-react'
import client from '../api/client.js'
import { fmtDateTime } from '../utils/date.js'

function MessageRow({ msg }) {
  const [expanded, setExpanded] = useState(false)
  return (
    <>
      <tr
        className="hover:bg-slate-50 cursor-pointer transition-colors"
        onClick={() => setExpanded(v => !v)}
      >
        <td className="px-5 py-3.5 text-slate-700 text-sm font-medium">{msg.from || msg.sender || '—'}</td>
        <td className="px-5 py-3.5 text-slate-800 text-sm">{msg.subject || '—'}</td>
        <td className="px-5 py-3.5 text-slate-500 text-xs">{msg.type || msg.message_type || '—'}</td>
        <td className="px-5 py-3.5 text-slate-400 text-xs">
          {fmtDateTime(msg.received_at || msg.created_at)}
        </td>
        <td className="px-5 py-3.5 text-slate-400">
          {expanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={5} className="px-5 pb-4 pt-0">
            <div className="bg-slate-50 rounded-xl p-4 text-sm text-slate-700 whitespace-pre-wrap">
              {msg.body || msg.content || msg.message || '(no content)'}
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

export default function MessagesPage() {
  const { t } = useTranslation()
  const [messages, setMessages] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    client.get('/v1/messages')
      .then(res => {
        const data = res.data
        setMessages(Array.isArray(data) ? data : (data?.messages || data?.items || []))
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 text-sm">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center gap-3 mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t('messages.title')}</h1>
        {messages.length > 0 && (
          <span className="bg-blue-100 text-blue-700 text-xs font-semibold px-2.5 py-1 rounded-full">
            {messages.length}
          </span>
        )}
      </div>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-xl px-4 py-3 text-sm">{error}</div>
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
                {[
                  t('messages.from'),
                  t('messages.subject'),
                  t('messages.type'),
                  t('messages.received_at'),
                  '',
                ].map((h, i) => (
                  <th key={i} className="text-left px-5 py-3.5 text-xs font-semibold text-slate-500 uppercase tracking-wide">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {messages.map((msg, i) => (
                <MessageRow key={msg.id || i} msg={msg} />
              ))}
              {messages.length === 0 && (
                <tr>
                  <td colSpan={5} className="py-16 text-center text-slate-400 text-sm">
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
