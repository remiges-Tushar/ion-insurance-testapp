import React from 'react'

const STAGES = [
  { key: 'va_created',    label: 'VA Created' },
  { key: 'payment_held',  label: 'Payment Held' },
  { key: 'policy_issued', label: 'Policy Issued' },
  { key: 'reconciling',   label: 'Reconciling' },
  { key: 'settled',       label: 'Settled' },
]

export default function SeamStageBar({ stage }) {
  const activeIdx = STAGES.findIndex(s => s.key === stage)
  return (
    <div className="flex items-center gap-1 w-full">
      {STAGES.map((s, i) => (
        <div key={s.key} className="flex-1 flex flex-col items-center gap-1">
          <div className={`h-1.5 w-full rounded-full ${i <= activeIdx ? 'bg-blue-500' : 'bg-gray-200'}`} />
          <span className={`text-[9px] font-medium ${i === activeIdx ? 'text-blue-600' : i < activeIdx ? 'text-green-600' : 'text-slate-400'}`}>
            {s.label}
          </span>
        </div>
      ))}
    </div>
  )
}
