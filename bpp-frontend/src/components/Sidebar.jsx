import React from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  LayoutDashboard, FileText, Package, Upload, BookOpen,
  AlertCircle, MessageSquare, LifeBuoy, Star, LogOut, Shield
} from 'lucide-react'

export default function Sidebar() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const links = [
    { to: '/overview', icon: LayoutDashboard, label: t('nav.overview') },
    { to: '/policies', icon: FileText, label: t('nav.policies') },
    { to: '/inventory', icon: Package, label: t('nav.inventory') },
    { to: '/publish', icon: Upload, label: t('nav.publish') },
    { to: '/catalogs', icon: BookOpen, label: t('nav.catalogs') },
    { to: '/claims', icon: AlertCircle, label: t('nav.claims') },
    { to: '/messages', icon: MessageSquare, label: t('nav.messages') },
    { to: '/support', icon: LifeBuoy, label: t('nav.support') },
    { to: '/ratings', icon: Star, label: t('nav.ratings') },
  ]

  function handleLogout() {
    localStorage.removeItem('bpp_token')
    navigate('/login')
  }

  return (
    <aside className="w-60 bg-slate-900 text-white flex flex-col">
      <div className="flex items-center gap-2 px-6 py-5 border-b border-slate-700">
        <Shield className="text-blue-400" size={24} />
        <span className="font-bold text-lg">BPP Admin</span>
      </div>
      <nav className="flex-1 py-4 space-y-1 px-3">
        {links.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                isActive
                  ? 'bg-blue-600 text-white'
                  : 'text-slate-300 hover:bg-slate-800 hover:text-white'
              }`
            }
          >
            <Icon size={18} />
            {label}
          </NavLink>
        ))}
      </nav>
      <div className="p-3 border-t border-slate-700">
        <button
          onClick={handleLogout}
          className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-300 hover:bg-slate-800 hover:text-white w-full transition-colors"
        >
          <LogOut size={18} />
          {t('nav.logout')}
        </button>
      </div>
    </aside>
  )
}
