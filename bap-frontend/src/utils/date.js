// Timestamps are stored UTC in the DB. Display in WIB (Asia/Jakarta, UTC+7).
const TZ = 'Asia/Jakarta'

export function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('id-ID', { timeZone: TZ, day: '2-digit', month: 'short', year: 'numeric' })
}

export function fmtDateTime(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('id-ID', {
    timeZone: TZ,
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit', hour12: false,
  }) + ' WIB'
}
