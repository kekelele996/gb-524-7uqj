import CheckCircleOutlineRounded from '@mui/icons-material/CheckCircleOutlineRounded'
import ErrorOutlineRounded from '@mui/icons-material/ErrorOutlineRounded'
import HighlightOffRounded from '@mui/icons-material/HighlightOffRounded'
import RemoveCircleOutlineRounded from '@mui/icons-material/RemoveCircleOutlineRounded'
import { Chip } from '@mui/material'
import type { ObservationQuality } from '../../types/observation'

const qualityLabels: Record<ObservationQuality, string> = {
  good: '良好',
  fair: '一般',
  poor: '偏低',
  excluded: '已排除'
}

export function QualityBadge({ quality }: { quality: ObservationQuality }) {
  const icon = quality === 'good'
    ? <CheckCircleOutlineRounded />
    : quality === 'fair'
      ? <RemoveCircleOutlineRounded />
      : quality === 'poor'
        ? <ErrorOutlineRounded />
        : <HighlightOffRounded />
  const color = quality === 'good' ? 'success' : quality === 'fair' ? 'info' : quality === 'poor' ? 'warning' : 'default'

  return (
    <Chip
      className={`quality-badge quality-${quality}`}
      icon={icon}
      label={qualityLabels[quality]}
      color={color}
      size="small"
      variant="outlined"
    />
  )
}

