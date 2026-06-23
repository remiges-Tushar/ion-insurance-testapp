const getHeaders = () => ({
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${localStorage.getItem('bpp_token') || ''}`
})

export const api = {
  get: (path) => fetch(path, { headers: getHeaders() }).then(r => r.json()),
  post: (path, body) => fetch(path, { method: 'POST', headers: getHeaders(), body: JSON.stringify(body) }).then(r => r.json()),
}
