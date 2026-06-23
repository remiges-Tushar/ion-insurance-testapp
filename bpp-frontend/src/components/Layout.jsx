import React from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  LayoutDashboard, FileText, Package, Upload,
  AlertCircle, MessageSquare, LifeBuoy, Star, LogOut, Shield
} from 'lucide-react'
import { motion } from 'framer-motion'

export default function Layout() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const navLinks = [
    { to: '/dashboard', icon: LayoutDashboard, label: t('nav.dashboard') },
    { to: '/policies', icon: FileText, label: t('nav.policies') },
    { to: '/inventory', icon: Package, label: t('nav.inventory') },
    { to: '/publish', icon: Upload, label: t('nav.publish') },
    { to: '/claims', icon: AlertCircle, label: t('nav.claims') },
    { to: '/messages', icon: MessageSquare, label: t('nav.messages') },
    { to: '/support', icon: LifeBuoy, label: t('nav.support') },
    { to: '/ratings', icon: Star, label: t('nav.ratings') },
  ]

  function handleLogout() {
    localStorage.removeItem('bpp_token')
    navigate('/login')
  }

  const email = localStorage.getItem('bpp_email') || 'admin@insurer.com'

  return (
    <div className="flex h-screen bg-slate-50 overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 bg-slate-900 text-white flex flex-col flex-shrink-0">
        {/* Logo */}
        <div className="flex items-center gap-3 px-6 py-5 border-b border-slate-700/60">
          <div className="w-9 h-9 bg-blue-500 rounded-lg flex items-center justify-center flex-shrink-0">
            <Shield size={20} className="text-white" />
          </div>
          <div>
            <p className="font-bold text-white text-sm leading-tight">BPP Insurance</p>
            <p className="text-slate-400 text-xs">Admin Dashboard</p>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 py-4 px-3 space-y-0.5 overflow-y-auto">
          {navLinks.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${
                  isActive
                    ? 'bg-blue-600 text-white shadow-sm'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <Icon size={18} className={isActive ? 'text-white' : 'text-slate-400'} />
                  {label}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* Footer: user info + logout */}
        <div className="p-3 border-t border-slate-700/60">
          <div className="px-3 py-2 mb-1">
            <p className="text-xs text-slate-400 truncate">{email}</p>
          </div>
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white w-full transition-all"
          >
            <LogOut size={18} />
            {t('nav.logout')}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        <div className="p-6 max-w-7xl mx-auto">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
