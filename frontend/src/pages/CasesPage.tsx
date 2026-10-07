import { FormEvent, useEffect, useMemo, useState } from 'react'
import AddRounded from '@mui/icons-material/AddRounded'
import ArrowForwardRounded from '@mui/icons-material/ArrowForwardRounded'
import GavelRounded from '@mui/icons-material/GavelRounded'
import LockRounded from '@mui/icons-material/LockRounded'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { PageHeader } from '../components/common/PageHeader'
import { ReviewDecisionDialog } from '../components/common/ReviewDecisionDialog'
import { useAuth } from '../hooks/useAuth'
import { useCaseStore } from '../stores/caseStore'
import { useLocalizationStore } from '../stores/localizationStore'
import type { CaseInput, CaseStatus, CaseSummary } from '../types/case'
import { formatDecimal, formatFrequency } from '../utils/format'

const statusLabel: Record<CaseStatus, string> = {
  draft: '草稿', collecting: '采集中', analyzing: '分析中', pending_review: '待复核', confirmed: '已确认', closed: '已关闭'
}

const initialCase: CaseInput = { case_code: '', title: '', frequency_center_hz: 433_920_000, priority: 'normal' }

export function CasesPage() {
  const { hasRole } = useAuth()
  const cases = useCaseStore((state) => state.cases)
  const load = useCaseStore((state) => state.load)
  const createCase = useCaseStore((state) => state.createCase)
  const transition = useCaseStore((state) => state.transition)
  const estimates = useLocalizationStore((state) => state.estimates)
  const loadEstimates = useLocalizationStore((state) => state.load)
  const [createOpen, setCreateOpen] = useState(false)
  const [reviewTarget, setReviewTarget] = useState<CaseSummary | null>(null)
  const [form, setForm] = useState<CaseInput>(initialCase)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    void Promise.all([load(), loadEstimates()])
  }, [load, loadEstimates])

  const latestByCase = useMemo(() => {
    const map = new Map<number, typeof estimates[number]>()
    estimates.forEach((estimate) => { if (!map.has(estimate.case_id)) map.set(estimate.case_id, estimate) })
    return map
  }, [estimates])

  const create = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try {
      await createCase(form)
      setCreateOpen(false)
      setForm(initialCase)
    } finally {
      setSaving(false)
    }
  }

  const advance = async (item: CaseSummary) => {
    const target: Partial<Record<CaseStatus, CaseStatus>> = { draft: 'collecting', collecting: 'analyzing', analyzing: 'pending_review', confirmed: 'closed' }
    const next = target[item.case_status]
    if (!next) return
    await transition(item.id, { target_status: next, version: item.version, conclusion: item.conclusion })
  }

  const canAdvance = (item: CaseSummary) => {
    if (item.case_status === 'draft') return hasRole('observer', 'analyst', 'admin')
    if (item.case_status === 'collecting' || item.case_status === 'analyzing') return hasRole('analyst', 'admin')
    if (item.case_status === 'confirmed') return hasRole('reviewer', 'admin')
    return false
  }

  return (
    <>
      <PageHeader
        eyebrow="CASE STATE / HUMAN REVIEW"
        title="定位案例流转"
        summary={`${cases.length} 个离线案例 · 确认仅限 reviewer · 关闭后所有观测与结论只读`}
        actions={hasRole('observer', 'analyst', 'admin') ? <Button variant="contained" startIcon={<AddRounded />} onClick={() => setCreateOpen(true)}>创建案例</Button> : undefined}
      />

      <section className="state-track" aria-label="案例状态流">
        {(['draft', 'collecting', 'analyzing', 'pending_review', 'confirmed', 'closed'] as CaseStatus[]).map((status, index) => (
          <div key={status} className="state-step"><span>{String(index + 1).padStart(2, '0')}</span><strong>{statusLabel[status]}</strong>{index < 5 && <ArrowForwardRounded />}</div>
        ))}
      </section>

      <section className="data-section" aria-labelledby="case-table-title">
        <Typography id="case-table-title" component="h2" variant="h6" mb={2}>案例、证据与乐观锁版本</Typography>
        <Box className="table-scroll">
          <Table size="small" aria-label="干扰案例列表">
            <TableHead><TableRow><TableCell>案例</TableCell><TableCell>中心频率</TableCell><TableCell>状态 / 版本</TableCell><TableCell>证据完整度</TableCell><TableCell>最新定位</TableCell><TableCell align="right">操作</TableCell></TableRow></TableHead>
            <TableBody>{cases.map((item) => {
              const estimate = latestByCase.get(item.id)
              return <TableRow key={item.id} hover>
                <TableCell><strong>{item.case_code}</strong><br /><span className="secondary-text">{item.title}</span></TableCell>
                <TableCell className="numeric">{formatFrequency(item.frequency_center_hz)}<br /><span className={`priority priority-${item.priority}`}>{item.priority === 'high' ? '高优先' : item.priority === 'low' ? '低优先' : '普通'}</span></TableCell>
                <TableCell><span className={`case-status case-${item.case_status}`}>{item.case_status === 'closed' && <LockRounded fontSize="inherit" />} {statusLabel[item.case_status]}</span><br /><span className="secondary-text">version {item.version}</span></TableCell>
                <TableCell className="numeric">有效观测 {item.active_observation_count} / {item.observation_count}<br />定位结果 {item.estimate_count}</TableCell>
                <TableCell>{estimate ? <><strong>残差 {formatDecimal(estimate.residual_deg)}°</strong><br /><span className="secondary-text">半径 {formatDecimal(estimate.uncertainty_radius_m, 0)} m</span></> : '尚未运行'}</TableCell>
                <TableCell align="right">
                  <Stack direction="row" justifyContent="flex-end" gap={1}>
                    {item.case_status === 'pending_review' && hasRole('reviewer', 'admin') && <Button size="small" variant="contained" startIcon={<GavelRounded />} onClick={() => setReviewTarget(item)}>复核</Button>}
                    {canAdvance(item) && <Button size="small" variant="outlined" onClick={() => void advance(item)}>{item.case_status === 'draft' ? '开始采集' : item.case_status === 'collecting' ? '进入分析' : item.case_status === 'analyzing' ? '提交复核' : '关闭案例'}</Button>}
                  </Stack>
                </TableCell>
              </TableRow>
            })}</TableBody>
          </Table>
        </Box>
      </section>

      <Dialog open={createOpen} onClose={saving ? undefined : () => setCreateOpen(false)} fullWidth maxWidth="sm">
        <form onSubmit={(event) => void create(event)}>
          <DialogTitle>创建离线干扰案例</DialogTitle>
          <DialogContent><Stack gap={2} sx={{ pt: 1 }}>
            <TextField label="案例编号" value={form.case_code} onChange={(event) => setForm({ ...form, case_code: event.target.value })} required />
            <TextField label="案例标题" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
            <TextField label="中心频率（Hz）" type="number" value={form.frequency_center_hz} onChange={(event) => setForm({ ...form, frequency_center_hz: Number(event.target.value) })} required />
            <TextField select label="优先级" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value as CaseInput['priority'] })}>
              <MenuItem value="low">低</MenuItem><MenuItem value="normal">普通</MenuItem><MenuItem value="high">高</MenuItem>
            </TextField>
          </Stack></DialogContent>
          <DialogActions><Button onClick={() => setCreateOpen(false)} disabled={saving}>继续查看</Button><Button type="submit" variant="contained" disabled={saving}>创建草稿案例</Button></DialogActions>
        </form>
      </Dialog>

      <ReviewDecisionDialog open={Boolean(reviewTarget)} item={reviewTarget} onClose={() => setReviewTarget(null)} onDecision={async (request) => { if (reviewTarget) await transition(reviewTarget.id, request) }} />
    </>
  )
}

