import { useEffect, useState } from 'react'
import GavelRounded from '@mui/icons-material/GavelRounded'
import ReplayRounded from '@mui/icons-material/ReplayRounded'
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material'
import type { CaseSummary, CaseTransition } from '../../types/case'

interface ReviewDecisionDialogProps {
  open: boolean
  item: CaseSummary | null
  onClose: () => void
  onDecision: (transition: CaseTransition) => Promise<void>
}

export function ReviewDecisionDialog({ open, item, onClose, onDecision }: ReviewDecisionDialogProps) {
  const [conclusion, setConclusion] = useState('')
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (open) {
      setConclusion(item?.conclusion ?? '')
      setReason('')
    }
  }, [open, item])

  if (!item) return null

  const decide = async (target: 'confirmed' | 'analyzing') => {
    setBusy(true)
    try {
      await onDecision({ target_status: target, version: item.version, conclusion, reason })
      onClose()
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth maxWidth="sm" aria-labelledby="review-dialog-title">
      <DialogTitle id="review-dialog-title">复核 {item.case_code}</DialogTitle>
      <DialogContent>
        <Stack gap={2} sx={{ pt: 1 }}>
          <Alert severity="warning" icon={<GavelRounded />}>定位坐标仅作为离线分析证据。确认操作不会触发执法、派工或设备控制。</Alert>
          <Typography variant="body2">有效观测 {item.active_observation_count} 条，定位结果 {item.estimate_count} 份。请结合残差、条件数和不确定半径独立判断。</Typography>
          <TextField
            label="人工复核结论"
            multiline minRows={3} value={conclusion}
            onChange={(event) => setConclusion(event.target.value)}
            helperText="确认案例时必填；说明接受结论所依据的证据。"
          />
          <TextField
            label="退回原因"
            multiline minRows={2} value={reason}
            onChange={(event) => setReason(event.target.value)}
            helperText="退回分析时至少填写 6 个字符。"
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>继续查看</Button>
        <Button startIcon={<ReplayRounded />} color="warning" onClick={() => void decide('analyzing')} disabled={busy || reason.trim().length < 6}>退回分析</Button>
        <Button startIcon={<GavelRounded />} variant="contained" onClick={() => void decide('confirmed')} disabled={busy || !conclusion.trim()}>确认定位结论</Button>
      </DialogActions>
    </Dialog>
  )
}

