import { Component, type ErrorInfo, type ReactNode, useEffect, useState } from 'react'
import { Alert, Button, CssBaseline, Snackbar, ThemeProvider, createTheme } from '@mui/material'
import { RouterProvider } from 'react-router-dom'
import { router } from './router'

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1d5d50', dark: '#173d35', light: '#dceae4', contrastText: '#f7fbf8' },
    secondary: { main: '#b86424', dark: '#7e431a', light: '#f5e5d6' },
    background: { default: '#f3f6f2', paper: '#fbfcf9' },
    text: { primary: '#18332c', secondary: '#557068' },
    divider: '#d5dfd8',
    warning: { main: '#aa5a1f' },
    error: { main: '#a23a32' },
    success: { main: '#2c6b4d' },
    info: { main: '#3e6c73' }
  },
  shape: { borderRadius: 6 },
  typography: {
    fontFamily: 'Aptos, "Source Han Sans SC", "Noto Sans CJK SC", "Microsoft YaHei", sans-serif',
    fontSize: 14,
    h4: { fontSize: '1.75rem', fontWeight: 700, lineHeight: 1.25, letterSpacing: 0 },
    h5: { fontSize: '1.375rem', fontWeight: 700, letterSpacing: 0 },
    h6: { fontSize: '1.1rem', fontWeight: 700, letterSpacing: 0 },
    button: { textTransform: 'none', fontWeight: 650, letterSpacing: 0 }
  },
  components: {
    MuiButton: { styleOverrides: { root: { minHeight: 40, boxShadow: 'none' } } },
    MuiIconButton: { styleOverrides: { root: { minWidth: 44, minHeight: 44 } } },
    MuiTableCell: { styleOverrides: { head: { fontWeight: 700, color: '#315047', backgroundColor: '#edf2ed' }, root: { borderColor: '#dce4de' } } },
    MuiDialog: { styleOverrides: { paper: { border: '1px solid #d5dfd8', boxShadow: '0 18px 48px rgba(23, 61, 53, 0.18)' } } },
    MuiChip: { styleOverrides: { root: { borderRadius: 4 } } }
  }
})

export function App() {
  const [error, setError] = useState('')

  useEffect(() => {
    const listener = (event: Event) => setError((event as CustomEvent<string>).detail)
    window.addEventListener('api:error', listener)
    return () => window.removeEventListener('api:error', listener)
  }, [])

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ErrorBoundary><RouterProvider router={router} /></ErrorBoundary>
      <Snackbar open={Boolean(error)} autoHideDuration={7000} onClose={() => setError('')} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        <Alert severity="error" variant="filled" onClose={() => setError('')}>{error}</Alert>
      </Snackbar>
    </ThemeProvider>
  )
}

interface BoundaryState { failed: boolean }

class ErrorBoundary extends Component<{ children: ReactNode }, BoundaryState> {
  state: BoundaryState = { failed: false }

  static getDerivedStateFromError(): BoundaryState { return { failed: true } }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('interface_error', error, info.componentStack)
  }

  render() {
    if (!this.state.failed) return this.props.children
    return <main className="fatal-state"><Alert severity="error">界面无法继续渲染。数据未被修改，请重新加载后继续。</Alert><Button variant="contained" onClick={() => window.location.reload()}>重新加载工作台</Button></main>
  }
}

