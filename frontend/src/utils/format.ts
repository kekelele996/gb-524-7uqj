export function formatFrequency(value: number): string {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(6)} GHz`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(6)} MHz`
  if (value >= 1_000) return `${(value / 1_000).toFixed(3)} kHz`
  return `${value.toFixed(0)} Hz`
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '未记录'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
  }).format(new Date(value))
}

export function formatCoordinate(value: number): string {
  return value.toFixed(6)
}

export function formatDecimal(value: number, digits = 2): string {
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits
  }).format(value)
}

