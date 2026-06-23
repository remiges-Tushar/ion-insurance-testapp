import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import RegisterPage from './pages/RegisterPage.jsx'
import OverviewPage from './pages/OverviewPage.jsx'
import PoliciesPage from './pages/PoliciesPage.jsx'
import PolicyDetailPage from './pages/PolicyDetailPage.jsx'
import InventoryPage from './pages/InventoryPage.jsx'
import PublishPage from './pages/PublishPage.jsx'
import ClaimsPage from './pages/ClaimsPage.jsx'
import MessagesPage from './pages/MessagesPage.jsx'
import SupportPage from './pages/SupportPage.jsx'
import RatingsPage from './pages/RatingsPage.jsx'
import Layout from './components/Layout.jsx'
import ProtectedRoute from './components/ProtectedRoute.jsx'

export default function App() {
  return (
    <Routes>
      {/* Public routes */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      {/* Protected routes wrapped in Layout */}
      <Route element={<ProtectedRoute />}>
        <Route element={<Layout />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<OverviewPage />} />
          <Route path="/policies" element={<PoliciesPage />} />
          <Route path="/policies/:id" element={<PolicyDetailPage />} />
          <Route path="/inventory" element={<InventoryPage />} />
          <Route path="/publish" element={<PublishPage />} />
          <Route path="/claims" element={<ClaimsPage />} />
          <Route path="/messages" element={<MessagesPage />} />
          <Route path="/support" element={<SupportPage />} />
          <Route path="/ratings" element={<RatingsPage />} />
        </Route>
      </Route>

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
