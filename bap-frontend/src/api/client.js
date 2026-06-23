import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json'
  }
})

export const discover = (q) => api.get('/discover', { params: { q } })

export const search = (query) => api.post('/discover', { query })

export const select = (body) => api.post('/select', body)

export const init = (body) => api.post('/init', body)

export const confirm = (body) => api.post('/confirm', body)

export const getStatus = (id) => api.get(`/status/${id}`)

export const requestStatus = (transaction_id) =>
  api.post('/request-status', { transaction_id })

export const cancel = (transaction_id) =>
  api.post('/cancel', { transaction_id })

export const rate = (body) => api.post('/rate', body)

export const support = (body) => api.post('/support', body)

export const getPolicies = () => api.get('/policies')

export const getPolicy = (id) => api.get(`/policies/${id}`)

export default api
