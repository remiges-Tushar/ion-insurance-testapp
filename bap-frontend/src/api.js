async function checkOk(r) {
  if (r.ok) return r.json()
  let msg
  try {
    const j = await r.json()
    msg = j?.detail || j?.message || j?.error || JSON.stringify(j)
  } catch {
    msg = await r.text().catch(() => `HTTP ${r.status}`)
  }
  throw new Error(msg || `HTTP ${r.status}`)
}

export const api = {
  get: (path) => fetch(path).then(checkOk),
  post: (path, body) => fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }).then(checkOk),
}
