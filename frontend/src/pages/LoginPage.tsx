import { FormEvent, useState } from 'react'
import LoginRounded from '@mui/icons-material/LoginRounded'
import RadarRounded from '@mui/icons-material/RadarRounded'
import { Alert, Box, Button, Stack, TextField, Typography } from '@mui/material'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const user = useAuthStore((state) => state.user)
  const login = useAuthStore((state) => state.login)
  const busy = useAuthStore((state) => state.busy)
  const [email, setEmail] = useState('analyst@spectrum.local')
  const [password, setPassword] = useState('Spectrum!2026')
  const [error, setError] = useState('')

  if (user) return <Navigate to="/stations" replace />

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    try {
      await login(email, password)
      const target = (location.state as { from?: string } | null)?.from ?? '/stations'
      navigate(target, { replace: true })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '登录失败，请检查账号和密码。')
    }
  }

  return (
    <main className="login-shell">
      <section className="login-signal" aria-label="离线方位线示意">
        <div className="signal-axis signal-axis-a" />
        <div className="signal-axis signal-axis-b" />
        <div className="signal-axis signal-axis-c" />
        <div className="signal-origin"><RadarRounded /></div>
        <Box className="login-brand">
          <Typography className="eyebrow">OFFLINE BEARING ANALYSIS</Typography>
          <Typography component="h1">频谱交汇分析台</Typography>
          <Typography>离线测向观测、几何证据与人工复核</Typography>
        </Box>
        <div className="safety-stamp">决策支持 / 非执法坐标</div>
      </section>

      <section className="login-form-section">
        <form onSubmit={(event) => void submit(event)} className="login-form">
          <Stack gap={3}>
            <Box>
              <Typography component="h2" variant="h5">登录工作区</Typography>
              <Typography color="text.secondary">使用授权身份进入案例与定位数据。</Typography>
            </Box>
            {error && <Alert severity="error">{error}</Alert>}
            <TextField label="邮箱" type="email" autoComplete="username" value={email} onChange={(event) => setEmail(event.target.value)} required fullWidth />
            <TextField label="密码" type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required fullWidth />
            <Button type="submit" variant="contained" size="large" startIcon={<LoginRounded />} disabled={busy}>
              {busy ? '正在验证身份' : '进入分析台'}
            </Button>
            <Box className="login-account-note">
              <Typography variant="caption">本地验收账号</Typography>
              <Typography variant="body2">analyst@spectrum.local / Spectrum!2026</Typography>
            </Box>
          </Stack>
        </form>
      </section>
    </main>
  )
}

