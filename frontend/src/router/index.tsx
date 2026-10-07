import { lazy, Suspense, useEffect, type ReactNode } from 'react'
import AccountTreeRounded from '@mui/icons-material/AccountTreeRounded'
import CellTowerRounded from '@mui/icons-material/CellTowerRounded'
import FactCheckRounded from '@mui/icons-material/FactCheckRounded'
import LogoutRounded from '@mui/icons-material/LogoutRounded'
import RadarRounded from '@mui/icons-material/RadarRounded'
import SensorsRounded from '@mui/icons-material/SensorsRounded'
import ShieldRounded from '@mui/icons-material/ShieldRounded'
import { Box, CircularProgress, IconButton, Tooltip, Typography } from '@mui/material'
import { createBrowserRouter, Navigate, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'
import type { UserRole } from '../types/auth'

const LoginPage = lazy(() => import('../pages/LoginPage').then((module) => ({ default: module.LoginPage })))
const StationsPage = lazy(() => import('../pages/StationsPage').then((module) => ({ default: module.StationsPage })))
const ObservationsPage = lazy(() => import('../pages/ObservationsPage').then((module) => ({ default: module.ObservationsPage })))
const LocalizationPage = lazy(() => import('../pages/LocalizationPage').then((module) => ({ default: module.LocalizationPage })))
const CasesPage = lazy(() => import('../pages/CasesPage').then((module) => ({ default: module.CasesPage })))
const AuditPage = lazy(() => import('../pages/AuditPage').then((module) => ({ default: module.AuditPage })))

interface NavigationItem {
  to: string
  label: string
  icon: typeof CellTowerRounded
  roles?: UserRole[]
}

const navigation: NavigationItem[] = [
  { to: '/stations', label: '测向站', icon: CellTowerRounded },
  { to: '/observations', label: '观测工作台', icon: SensorsRounded },
  { to: '/localization', label: '三角定位', icon: RadarRounded },
  { to: '/cases', label: '案例流转', icon: AccountTreeRounded },
  { to: '/audit', label: '审计追踪', icon: FactCheckRounded, roles: ['reviewer', 'admin'] }
]

function AuthGate() {
  const user = useAuthStore((state) => state.user)
  const initialized = useAuthStore((state) => state.initialized)
  const loadMe = useAuthStore((state) => state.loadMe)
  const logout = useAuthStore((state) => state.logout)
  const location = useLocation()

  useEffect(() => { void loadMe() }, [loadMe])
  useEffect(() => {
    const expired = () => logout()
    window.addEventListener('auth:expired', expired)
    return () => window.removeEventListener('auth:expired', expired)
  }, [logout])

  if (!initialized) return <main className="loading-screen"><CircularProgress size={30} /><Typography>正在验证会话</Typography></main>
  if (!user) return <Navigate to="/login" state={{ from: location.pathname }} replace />
  return <AppShell />
}

function RoleGate({ roles }: { roles: UserRole[] }) {
  const role = useAuthStore((state) => state.user?.role)
  if (!role || !roles.includes(role)) return <Navigate to="/stations" replace />
  return <Outlet />
}

function AppShell() {
  const user = useAuthStore((state) => state.user)!
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()

  const signOut = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">跳到主要内容</a>
      <aside className="app-sidebar">
        <div className="brand-lockup"><RadarRounded /><div><strong>频谱交汇</strong><span>离线分析台</span></div></div>
        <nav aria-label="主导航">
          {navigation.filter((item) => !item.roles || item.roles.includes(user.role)).map((item) => {
            const Icon = item.icon
            return <NavLink key={item.to} to={item.to}><Icon /><span>{item.label}</span></NavLink>
          })}
        </nav>
        <div className="sidebar-safety"><ShieldRounded /><span>定位结论须人工复核<br />不会下发设备或执法动作</span></div>
      </aside>
      <div className="app-workspace">
        <header className="app-topbar">
          <div className="mobile-brand"><RadarRounded /><strong>频谱交汇</strong></div>
          <div className="topbar-context"><span className="live-dot" />离线分析环境</div>
          <div className="user-context"><div><strong>{user.display_name}</strong><span>{roleLabel(user.role)}</span></div><Tooltip title="退出登录"><IconButton aria-label="退出登录" onClick={signOut}><LogoutRounded /></IconButton></Tooltip></div>
        </header>
        <main id="main-content" className="app-content"><Outlet /></main>
      </div>
    </div>
  )
}

function roleLabel(role: UserRole) {
  return { observer: '观测员', analyst: '分析员', reviewer: '复核员', admin: '管理员' }[role]
}

function LazyPage({ children }: { children: ReactNode }) {
  return <Suspense fallback={<main className="loading-screen"><CircularProgress size={28} /><Typography>正在载入工作区</Typography></main>}>{children}</Suspense>
}

export const router = createBrowserRouter([
  { path: '/login', element: <LazyPage><LoginPage /></LazyPage> },
  {
    path: '/', element: <AuthGate />, children: [
      { index: true, element: <Navigate to="/stations" replace /> },
      { path: 'stations', element: <LazyPage><StationsPage /></LazyPage> },
      { path: 'observations', element: <LazyPage><ObservationsPage /></LazyPage> },
      { path: 'localization', element: <LazyPage><LocalizationPage /></LazyPage> },
      { path: 'cases', element: <LazyPage><CasesPage /></LazyPage> },
      { element: <RoleGate roles={['reviewer', 'admin']} />, children: [{ path: 'audit', element: <LazyPage><AuditPage /></LazyPage> }] }
    ]
  },
  { path: '*', element: <Navigate to="/" replace /> }
])
