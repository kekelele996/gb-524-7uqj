import { FormEvent, useEffect, useMemo, useState } from 'react'
import AddRounded from '@mui/icons-material/AddRounded'
import CalibrationRounded from '@mui/icons-material/CompassCalibrationRounded'
import EditRounded from '@mui/icons-material/EditRounded'
import { Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, Snackbar, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { BearingPlot } from '../components/common/BearingPlot'
import { PageHeader } from '../components/common/PageHeader'
import { useAuth } from '../hooks/useAuth'
import { useObservationStore } from '../stores/observationStore'
import { useStationStore } from '../stores/stationStore'
import type { ReceiverStation, StationInput, StationReintersection } from '../types/station'
import { formatCoordinate, formatDateTime, formatDecimal } from '../utils/format'

const initialStation: StationInput = {
  station_code: '', name: '', latitude: 31.2304, longitude: 121.4737,
  antenna_bias_deg: 0, accuracy_deg: 1.5, station_status: 'active', calibrated_at: new Date().toISOString()
}

function stationToForm(station: ReceiverStation): StationInput {
  return {
    station_code: station.station_code, name: station.name, latitude: station.latitude, longitude: station.longitude,
    antenna_bias_deg: station.antenna_bias_deg, accuracy_deg: station.accuracy_deg,
    station_status: station.station_status, calibrated_at: station.calibrated_at
  }
}

export function StationsPage() {
  const { hasRole } = useAuth()
  const stations = useStationStore((state) => state.stations)
  const loadStations = useStationStore((state) => state.load)
  const createStation = useStationStore((state) => state.createStation)
  const updateStation = useStationStore((state) => state.updateStation)
  const busy = useStationStore((state) => state.busy)
  const observations = useObservationStore((state) => state.observations)
  const loadObservations = useObservationStore((state) => state.load)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<ReceiverStation | null>(null)
  const [form, setForm] = useState<StationInput>(initialStation)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [reintersection, setReintersection] = useState<StationReintersection | null>(null)

  useEffect(() => {
    void Promise.all([loadStations(), loadObservations()])
  }, [loadStations, loadObservations])

  const activeCount = useMemo(() => stations.filter((station) => station.station_status === 'active').length, [stations])
  const canManage = hasRole('analyst', 'admin')

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try {
      await createStation(form)
      setOpen(false)
      setForm({ ...initialStation, calibrated_at: new Date().toISOString() })
    } finally {
      setSaving(false)
    }
  }

  const submitEdit = async (event: FormEvent) => {
    event.preventDefault()
    if (!editing) return
    setSaving(true)
    try {
      const result = await updateStation(editing.id, {
        name: form.name, latitude: form.latitude, longitude: form.longitude,
        antenna_bias_deg: form.antenna_bias_deg, accuracy_deg: form.accuracy_deg,
        station_status: form.station_status, calibrated_at: form.calibrated_at
      })
      setEditing(null)
      setReintersection(result.reintersection)
      if (result.reintersection?.triggered) {
        setNotice(`站点已保存，${result.reintersection.estimates_created} 个未结案案例已按当前站点值重新交汇。`)
      } else {
        setNotice('站点校准信息已保存。')
      }
    } finally {
      setSaving(false)
    }
  }

  const startEdit = (station: ReceiverStation) => {
    setEditing(station)
    setForm(stationToForm(station))
  }

  return (
    <>
      <PageHeader
        eyebrow="RECEIVER GEOMETRY / CALIBRATION"
        title="测向站与观测覆盖"
        summary={`${stations.length} 个测向站 · ${activeCount} 个可参与定位 · 坐标仅用于离线局部平面计算`}
        actions={canManage ? <Button variant="contained" startIcon={<AddRounded />} onClick={() => setOpen(true)}>登记测向站</Button> : undefined}
      />

      <section className="plot-section">
        <BearingPlot stations={stations} observations={observations.slice(0, 20)} height={390} />
      </section>

      {reintersection?.triggered && (
        <Alert severity={reintersection.skipped_cases.length > 0 ? 'warning' : 'success'} sx={{ mb: 2 }}
          onClose={() => setReintersection(null)}>
          {reintersection.estimates_created > 0 &&
            <span>已重新交汇案例：{reintersection.open_cases.map((item) => item.case_code).join('、')}。</span>}
          {reintersection.skipped_cases.length > 0 &&
            <span> 未形成新定位点：{reintersection.skipped_cases.map((item) => `${item.case_code}（${item.reason}）`).join('、')}；校正方位仍已刷新。</span>}
          {' '}已确认或关闭的案例保持原结论不变。
        </Alert>
      )}

      <section className="data-section" aria-labelledby="station-table-title">
        <Stack direction="row" justifyContent="space-between" alignItems="baseline" mb={2}>
          <Typography id="station-table-title" component="h2" variant="h6">站点校准台账</Typography>
          <Typography variant="body2" color="text.secondary">保存校准后，未结案案例按当前站点值重新交汇</Typography>
        </Stack>
        <Box className="table-scroll">
          <Table size="small" aria-label="测向站列表">
            <TableHead><TableRow><TableCell>站点</TableCell><TableCell>WGS84 坐标</TableCell><TableCell>精度 / 偏置</TableCell><TableCell>状态</TableCell><TableCell>最近校准</TableCell>{canManage && <TableCell align="right">校准操作</TableCell>}</TableRow></TableHead>
            <TableBody>
              {stations.map((station) => (
                <TableRow key={station.id} hover>
                  <TableCell><strong>{station.station_code}</strong><br /><span className="secondary-text">{station.name}</span></TableCell>
                  <TableCell className="numeric">{formatCoordinate(station.latitude)}<br />{formatCoordinate(station.longitude)}</TableCell>
                  <TableCell className="numeric">±{formatDecimal(station.accuracy_deg, 1)}° / {station.antenna_bias_deg >= 0 ? '+' : ''}{formatDecimal(station.antenna_bias_deg, 1)}°</TableCell>
                  <TableCell><span className={`status-text status-${station.station_status}`}>{station.station_status === 'active' ? '● 已启用' : station.station_status === 'calibration_due' ? '△ 待校准' : '○ 已停用'}</span></TableCell>
                  <TableCell>{formatDateTime(station.calibrated_at)}</TableCell>
                  {canManage && (
                    <TableCell align="right">
                      <Button size="small" startIcon={<EditRounded />} onClick={() => startEdit(station)}>现场复检</Button>
                    </TableCell>
                  )}
                </TableRow>
              ))}
              {!busy && stations.length === 0 && <TableRow><TableCell colSpan={canManage ? 6 : 5}>尚无测向站。登记并校准站点后才能录入方位观测。</TableCell></TableRow>}
            </TableBody>
          </Table>
        </Box>
      </section>

      <Dialog open={open} onClose={saving ? undefined : () => setOpen(false)} fullWidth maxWidth="sm">
        <form onSubmit={(event) => void submit(event)}>
          <DialogTitle>登记离线测向站</DialogTitle>
          <DialogContent>
            <Stack gap={2} sx={{ pt: 1 }}>
              <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                <TextField label="站点编号" value={form.station_code} onChange={(event) => setForm({ ...form, station_code: event.target.value })} required fullWidth />
                <TextField label="站点名称" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required fullWidth />
              </Stack>
              <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                <TextField label="纬度" type="number" inputProps={{ step: '0.000001' }} value={form.latitude} onChange={(event) => setForm({ ...form, latitude: Number(event.target.value) })} required fullWidth />
                <TextField label="经度" type="number" inputProps={{ step: '0.000001' }} value={form.longitude} onChange={(event) => setForm({ ...form, longitude: Number(event.target.value) })} required fullWidth />
              </Stack>
              <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                <TextField label="精度（度）" type="number" inputProps={{ step: '0.1', min: 0.1, max: 45 }} value={form.accuracy_deg} onChange={(event) => setForm({ ...form, accuracy_deg: Number(event.target.value) })} required fullWidth />
                <TextField label="天线偏置（度）" type="number" inputProps={{ step: '0.1', min: -30, max: 30 }} value={form.antenna_bias_deg} onChange={(event) => setForm({ ...form, antenna_bias_deg: Number(event.target.value) })} fullWidth />
              </Stack>
              <TextField select label="站点状态" value={form.station_status} onChange={(event) => setForm({ ...form, station_status: event.target.value as StationInput['station_status'] })}>
                <MenuItem value="active">已启用</MenuItem><MenuItem value="calibration_due">待校准</MenuItem><MenuItem value="inactive">已停用</MenuItem>
              </TextField>
              <TextField label="校准时间" type="datetime-local" value={form.calibrated_at?.slice(0, 16) ?? ''} onChange={(event) => setForm({ ...form, calibrated_at: event.target.value ? new Date(event.target.value).toISOString() : null })} InputLabelProps={{ shrink: true }} />
            </Stack>
          </DialogContent>
          <DialogActions><Button onClick={() => setOpen(false)} disabled={saving}>继续查看</Button><Button type="submit" variant="contained" startIcon={<CalibrationRounded />} disabled={saving}>保存校准站点</Button></DialogActions>
        </form>
      </Dialog>

      <Dialog open={editing !== null} onClose={saving ? undefined : () => setEditing(null)} fullWidth maxWidth="sm">
        <form onSubmit={(event) => void submitEdit(event)}>
          <DialogTitle>现场复检校准 · {editing?.station_code}</DialogTitle>
          <DialogContent>
            <Typography variant="body2" color="text.secondary" sx={{ pt: 1, pb: 2 }}>
              保存后，该站点参与且尚未结案的案例会用当前坐标、精度和天线偏置重新交汇；已确认或关闭的案例维持原样。
            </Typography>
            <Stack gap={2}>
              <TextField label="站点名称" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required fullWidth />
              <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                <TextField label="纬度" type="number" inputProps={{ step: '0.000001' }} value={form.latitude} onChange={(event) => setForm({ ...form, latitude: Number(event.target.value) })} required fullWidth />
                <TextField label="经度" type="number" inputProps={{ step: '0.000001' }} value={form.longitude} onChange={(event) => setForm({ ...form, longitude: Number(event.target.value) })} required fullWidth />
              </Stack>
              <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                <TextField label="精度（度）" type="number" inputProps={{ step: '0.1', min: 0.1, max: 45 }} value={form.accuracy_deg} onChange={(event) => setForm({ ...form, accuracy_deg: Number(event.target.value) })} required fullWidth />
                <TextField label="天线偏置（度）" type="number" inputProps={{ step: '0.1', min: -30, max: 30 }} value={form.antenna_bias_deg} onChange={(event) => setForm({ ...form, antenna_bias_deg: Number(event.target.value) })} fullWidth />
              </Stack>
              <TextField select label="站点状态" value={form.station_status} onChange={(event) => setForm({ ...form, station_status: event.target.value as StationInput['station_status'] })}>
                <MenuItem value="active">已启用</MenuItem><MenuItem value="calibration_due">待校准</MenuItem><MenuItem value="inactive">已停用</MenuItem>
              </TextField>
              <TextField label="校准时间" type="datetime-local" value={form.calibrated_at?.slice(0, 16) ?? ''} onChange={(event) => setForm({ ...form, calibrated_at: event.target.value ? new Date(event.target.value).toISOString() : null })} InputLabelProps={{ shrink: true }} />
            </Stack>
          </DialogContent>
          <DialogActions><Button onClick={() => setEditing(null)} disabled={saving}>继续查看</Button><Button type="submit" variant="contained" startIcon={<CalibrationRounded />} disabled={saving}>保存并重新交汇</Button></DialogActions>
        </form>
      </Dialog>

      <Snackbar open={notice !== null} autoHideDuration={5000} onClose={() => setNotice(null)} message={notice ?? ''} />
    </>
  )
}
