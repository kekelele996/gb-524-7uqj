import { useEffect, useMemo, useState } from 'react'
import PlayArrowRounded from '@mui/icons-material/PlayArrowRounded'
import ScienceRounded from '@mui/icons-material/ScienceRounded'
import { Alert, Box, Button, FormControlLabel, MenuItem, Stack, Switch, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { BearingPlot } from '../components/common/BearingPlot'
import { PageHeader } from '../components/common/PageHeader'
import { QualityBadge } from '../components/common/QualityBadge'
import { useAuth } from '../hooks/useAuth'
import { useLocalizationRun } from '../hooks/useLocalizationRun'
import { useCaseStore } from '../stores/caseStore'
import { useLocalizationStore } from '../stores/localizationStore'
import { useObservationStore } from '../stores/observationStore'
import { useStationStore } from '../stores/stationStore'
import type { LocalizationEstimate } from '../types/localization'
import { formatCoordinate, formatDateTime, formatDecimal, formatFrequency } from '../utils/format'

export function LocalizationPage() {
  const { hasRole } = useAuth()
  const cases = useCaseStore((state) => state.cases)
  const loadCases = useCaseStore((state) => state.load)
  const stations = useStationStore((state) => state.stations)
  const loadStations = useStationStore((state) => state.load)
  const observations = useObservationStore((state) => state.observations)
  const loadObservations = useObservationStore((state) => state.load)
  const estimates = useLocalizationStore((state) => state.estimates)
  const selected = useLocalizationStore((state) => state.selected)
  const select = useLocalizationStore((state) => state.select)
  const loadEstimates = useLocalizationStore((state) => state.load)
  const { execute, busy, lastResult } = useLocalizationRun()
  const [caseId, setCaseId] = useState(0)
  const [allowOutlier, setAllowOutlier] = useState(true)

  useEffect(() => {
    void Promise.all([loadCases(), loadStations()])
  }, [loadCases, loadStations])

  useEffect(() => {
    if (!caseId) {
      const analyzing = cases.find((item) => item.case_status === 'analyzing')
      if (analyzing) setCaseId(analyzing.id)
    }
  }, [caseId, cases])

  useEffect(() => {
    if (caseId) void Promise.all([loadObservations(caseId), loadEstimates(caseId)])
  }, [caseId, loadEstimates, loadObservations])

  const selectedCase = cases.find((item) => item.id === caseId)
  const residuals = selected?.residuals_json ?? []
  const observationById = useMemo(() => new Map(observations.map((item) => [item.id, item])), [observations])

  const run = async () => {
    if (!caseId) return
    await execute(caseId, allowOutlier)
    await loadCases()
  }

  return (
    <>
      <PageHeader
        eyebrow="WEIGHTED BEARING INTERSECTION / WLS V1"
        title="三角定位证据台"
        summary={selectedCase ? `${selectedCase.case_code} · ${formatFrequency(selectedCase.frequency_center_hz)} · ${selectedCase.active_observation_count} 条有效观测` : '选择 analyzing 案例后运行离线定位'}
        actions={hasRole('analyst', 'admin') ? <Button variant="contained" startIcon={<PlayArrowRounded />} disabled={!caseId || selectedCase?.case_status !== 'analyzing' || busy} onClick={() => void run()}>{busy ? '正在计算' : '运行加权定位'}</Button> : undefined}
      />

      <section className="control-strip localization-controls">
        <TextField select size="small" label="分析案例" value={caseId || ''} onChange={(event) => setCaseId(Number(event.target.value))} sx={{ minWidth: 330 }}>
          {cases.filter((item) => item.case_status === 'analyzing' || item.id === caseId).map((item) => <MenuItem key={item.id} value={item.id}>{item.case_code} · {item.title}</MenuItem>)}
        </TextField>
        <FormControlLabel control={<Switch checked={allowOutlier} onChange={(event) => setAllowOutlier(event.target.checked)} />} label="生成可解释离群候选" />
        {lastResult?.candidate && <Alert severity="warning">已保留原估计，并生成剔除观测 #{lastResult.candidate.outlier_ids_json[0]} 的候选重算。</Alert>}
      </section>

      <Alert severity="info" icon={<ScienceRounded />} className="safety-alert">估计坐标、不确定半径和离群候选均为离线模型证据，必须与原始方位线和残差共同复核。</Alert>

      <section className="localization-grid">
        <div className="plot-section plot-primary">
          <BearingPlot stations={stations} observations={observations} estimate={selected} height={520} />
        </div>
        <aside className="estimate-rail" aria-label="定位结果历史">
          <Typography component="h2" variant="h6">不可覆盖的运行历史</Typography>
          <Stack gap={1.5} mt={2}>
            {estimates.map((estimate) => <EstimateButton key={estimate.id} estimate={estimate} selected={selected?.id === estimate.id} onClick={() => select(estimate)} />)}
            {estimates.length === 0 && <Typography color="text.secondary">尚无运行结果。有效观测满足几何条件后可运行定位。</Typography>}
          </Stack>
        </aside>
      </section>

      <section className="data-section" aria-labelledby="residual-title">
        <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" gap={1} mb={2}>
          <Typography id="residual-title" component="h2" variant="h6">逐站角度残差</Typography>
          {selected && <Typography variant="body2" color="text.secondary">算法 {selected.algorithm_version} · 创建于 {formatDateTime(selected.created_at)}</Typography>}
        </Stack>
        <Box className="table-scroll">
          <Table size="small" aria-label="定位残差证据">
            <TableHead><TableRow><TableCell>观测 / 测向站</TableCell><TableCell>观测方位</TableCell><TableCell>预测方位</TableCell><TableCell>角度残差</TableCell><TableCell>标准化残差</TableCell><TableCell>质量</TableCell></TableRow></TableHead>
            <TableBody>
              {residuals.map((residual) => {
                const observation = observationById.get(residual.observation_id)
                return <TableRow key={residual.observation_id} className={selected?.outlier_ids_json.includes(residual.observation_id) ? 'row-warning' : ''}>
                  <TableCell><strong>#{residual.observation_id}</strong> · {residual.station_code}</TableCell>
                  <TableCell className="numeric">{formatDecimal(residual.observed_deg, 2)}°</TableCell>
                  <TableCell className="numeric">{formatDecimal(residual.predicted_deg, 2)}°</TableCell>
                  <TableCell className="numeric">{residual.residual_deg >= 0 ? '+' : ''}{formatDecimal(residual.residual_deg, 2)}°</TableCell>
                  <TableCell className="numeric">{formatDecimal(residual.standardized, 2)} σ {selected?.outlier_ids_json.includes(residual.observation_id) && <strong> · 离群证据</strong>}</TableCell>
                  <TableCell>{observation ? <QualityBadge quality={observation.quality} /> : '历史快照'}</TableCell>
                </TableRow>
              })}
              {!selected && <TableRow><TableCell colSpan={6}>选择或运行一条定位结果后显示逐站残差。</TableCell></TableRow>}
            </TableBody>
          </Table>
        </Box>
      </section>
    </>
  )
}

function EstimateButton({ estimate, selected, onClick }: { estimate: LocalizationEstimate; selected: boolean; onClick: () => void }) {
  return (
    <button type="button" className={`estimate-item ${selected ? 'is-selected' : ''}`} onClick={onClick}>
      <span className="estimate-item-top"><strong>运行 #{estimate.id}</strong><span>{estimate.estimate_status === 'outlier_candidate' ? '△ 离群候选' : '◆ 原始估计'}</span></span>
      <span className="estimate-coordinate">{formatCoordinate(estimate.latitude)}, {formatCoordinate(estimate.longitude)}</span>
      <span className="estimate-metrics">残差 {formatDecimal(estimate.residual_deg)}° · 半径 {formatDecimal(estimate.uncertainty_radius_m, 0)} m</span>
    </button>
  )
}

