import { FormEvent, useEffect, useMemo, useState } from 'react'
import AddRounded from '@mui/icons-material/AddRounded'
import BlockRounded from '@mui/icons-material/BlockRounded'
import FactCheckRounded from '@mui/icons-material/FactCheckRounded'
import { Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { PageHeader } from '../components/common/PageHeader'
import { QualityBadge } from '../components/common/QualityBadge'
import { useAuth } from '../hooks/useAuth'
import { useCaseStore } from '../stores/caseStore'
import { useObservationStore } from '../stores/observationStore'
import { useStationStore } from '../stores/stationStore'
import type { BearingObservation, ObservationInput } from '../types/observation'
import { formatDateTime, formatDecimal, formatFrequency } from '../utils/format'

export function ObservationsPage() {
  const { hasRole } = useAuth()
  const cases = useCaseStore((state) => state.cases)
  const loadCases = useCaseStore((state) => state.load)
  const stations = useStationStore((state) => state.stations)
  const loadStations = useStationStore((state) => state.load)
  const observations = useObservationStore((state) => state.observations)
  const loadObservations = useObservationStore((state) => state.load)
  const createObservation = useObservationStore((state) => state.createObservation)
  const excludeObservation = useObservationStore((state) => state.excludeObservation)
  const validateCase = useObservationStore((state) => state.validateCase)
  const validation = useObservationStore((state) => state.validation)
  const [caseId, setCaseId] = useState(0)
  const [createOpen, setCreateOpen] = useState(false)
  const [excludeTarget, setExcludeTarget] = useState<BearingObservation | null>(null)
  const [excludeReason, setExcludeReason] = useState('')
  const [saving, setSaving] = useState(false)
  const selectedCase = cases.find((item) => item.id === caseId)
  const [form, setForm] = useState<ObservationInput>({ station_id: 0, case_id: 0, bearing_deg: 0, signal_dbm: -70, frequency_hz: 433_920_000, bandwidth_hz: 12_500, quality: 'good' })

  useEffect(() => {
    void Promise.all([loadCases(), loadStations()])
  }, [loadCases, loadStations])

  useEffect(() => {
    if (!caseId && cases[0]) setCaseId(cases[0].id)
  }, [caseId, cases])

  useEffect(() => {
    if (!caseId) return
    void loadObservations(caseId)
    const item = cases.find((candidate) => candidate.id === caseId)
    if (item) setForm((current) => ({ ...current, case_id: item.id, frequency_hz: item.frequency_center_hz }))
  }, [caseId, cases, loadObservations])

  useEffect(() => {
    if (!form.station_id && stations[0]) setForm((current) => ({ ...current, station_id: stations[0].id }))
  }, [form.station_id, stations])

  const activeObservations = useMemo(() => observations.filter((item) => item.quality !== 'excluded').length, [observations])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try {
      await createObservation(form)
      setCreateOpen(false)
    } finally {
      setSaving(false)
    }
  }

  const exclude = async () => {
    if (!excludeTarget) return
    setSaving(true)
    try {
      await excludeObservation(excludeTarget.id, excludeReason)
      setExcludeTarget(null)
      setExcludeReason('')
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <PageHeader
        eyebrow="BEARING INTAKE / QUALITY CONTROL"
        title="观测工作台"
        summary={selectedCase ? `${selectedCase.case_code} · ${formatFrequency(selectedCase.frequency_center_hz)} · ${activeObservations} 条有效观测` : '选择案例后录入观测'}
        actions={
          <>
            <Button variant="outlined" startIcon={<FactCheckRounded />} disabled={!caseId} onClick={() => void validateCase(caseId)}>批量校验</Button>
            {hasRole('observer', 'analyst', 'admin') && <Button variant="contained" startIcon={<AddRounded />} disabled={!caseId} onClick={() => setCreateOpen(true)}>录入观测</Button>}
          </>
        }
      />

      <section className="control-strip">
        <TextField select size="small" label="当前案例" value={caseId || ''} onChange={(event) => setCaseId(Number(event.target.value))} sx={{ minWidth: 320 }}>
          {cases.filter((item) => item.case_status !== 'closed').map((item) => <MenuItem key={item.id} value={item.id}>{item.case_code} · {item.title}</MenuItem>)}
        </TextField>
        {validation && <Alert severity={validation.invalid ? 'warning' : 'success'}>{validation.valid} 条可用，{validation.invalid} 条需处理</Alert>}
      </section>

      <section className="data-section" aria-labelledby="observation-table-title">
        <Typography id="observation-table-title" component="h2" variant="h6" mb={2}>原始值与天线偏置校正</Typography>
        <Box className="table-scroll">
          <Table size="small" aria-label="方位观测列表">
            <TableHead><TableRow><TableCell>测向站 / 时间</TableCell><TableCell>原始 → 校正方位</TableCell><TableCell>频率 / 带宽</TableCell><TableCell>信号</TableCell><TableCell>质量</TableCell><TableCell align="right">操作</TableCell></TableRow></TableHead>
            <TableBody>
              {observations.map((item) => (
                <TableRow key={item.id} hover className={item.quality === 'excluded' ? 'row-muted' : ''}>
                  <TableCell><strong>{item.station?.station_code ?? `站点 #${item.station_id}`}</strong><br /><span className="secondary-text">{formatDateTime(item.observed_at)}</span></TableCell>
                  <TableCell className="numeric">{formatDecimal(item.bearing_deg, 1)}° → <strong>{formatDecimal(item.corrected_bearing_deg, 1)}°</strong></TableCell>
                  <TableCell className="numeric">{formatFrequency(item.frequency_hz)}<br /><span className="secondary-text">BW {formatFrequency(item.bandwidth_hz)}</span></TableCell>
                  <TableCell className="numeric">{formatDecimal(item.signal_dbm, 1)} dBm</TableCell>
                  <TableCell><QualityBadge quality={item.quality} />{item.excluded_reason && <Typography variant="caption" display="block">{item.excluded_reason}</Typography>}</TableCell>
                  <TableCell align="right">{hasRole('analyst', 'admin') && item.quality !== 'excluded' && <Button size="small" color="warning" startIcon={<BlockRounded />} onClick={() => setExcludeTarget(item)}>排除</Button>}</TableCell>
                </TableRow>
              ))}
              {observations.length === 0 && <TableRow><TableCell colSpan={6}>当前案例没有观测。至少录入来自两个启用站点的方位才能运行定位。</TableCell></TableRow>}
            </TableBody>
          </Table>
        </Box>
      </section>

      <Dialog open={createOpen} onClose={saving ? undefined : () => setCreateOpen(false)} fullWidth maxWidth="sm">
        <form onSubmit={(event) => void submit(event)}>
          <DialogTitle>录入离线方位观测</DialogTitle>
          <DialogContent><Stack gap={2} sx={{ pt: 1 }}>
            <TextField select label="测向站" value={form.station_id || ''} onChange={(event) => setForm({ ...form, station_id: Number(event.target.value) })} required>
              {stations.filter((item) => item.station_status === 'active').map((item) => <MenuItem key={item.id} value={item.id}>{item.station_code} · ±{item.accuracy_deg}°</MenuItem>)}
            </TextField>
            <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
              <TextField label="原始方位（度）" type="number" inputProps={{ min: 0, max: 359.999, step: 0.1 }} value={form.bearing_deg} onChange={(event) => setForm({ ...form, bearing_deg: Number(event.target.value) })} required fullWidth />
              <TextField select label="质量" value={form.quality} onChange={(event) => setForm({ ...form, quality: event.target.value as ObservationInput['quality'] })} fullWidth>
                <MenuItem value="good">良好</MenuItem><MenuItem value="fair">一般</MenuItem><MenuItem value="poor">偏低</MenuItem>
              </TextField>
            </Stack>
            <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
              <TextField label="频率（Hz）" type="number" value={form.frequency_hz} onChange={(event) => setForm({ ...form, frequency_hz: Number(event.target.value) })} required fullWidth />
              <TextField label="带宽（Hz）" type="number" value={form.bandwidth_hz} onChange={(event) => setForm({ ...form, bandwidth_hz: Number(event.target.value) })} required fullWidth />
            </Stack>
            <TextField label="信号强度（dBm）" type="number" inputProps={{ min: -200, max: 50, step: 0.1 }} value={form.signal_dbm} onChange={(event) => setForm({ ...form, signal_dbm: Number(event.target.value) })} required />
          </Stack></DialogContent>
          <DialogActions><Button onClick={() => setCreateOpen(false)} disabled={saving}>继续查看</Button><Button type="submit" variant="contained" disabled={saving}>保存观测</Button></DialogActions>
        </form>
      </Dialog>

      <Dialog open={Boolean(excludeTarget)} onClose={saving ? undefined : () => setExcludeTarget(null)} fullWidth maxWidth="xs">
        <DialogTitle>排除观测 #{excludeTarget?.id}</DialogTitle>
        <DialogContent><TextField sx={{ mt: 1 }} fullWidth multiline minRows={3} label="排除证据" value={excludeReason} onChange={(event) => setExcludeReason(event.target.value)} helperText="该动作保留原始值并写入审计，至少填写 6 个字符。" /></DialogContent>
        <DialogActions><Button onClick={() => setExcludeTarget(null)} disabled={saving}>保留观测</Button><Button color="warning" variant="contained" disabled={saving || excludeReason.trim().length < 6} onClick={() => void exclude()}>记录并排除</Button></DialogActions>
      </Dialog>
    </>
  )
}

