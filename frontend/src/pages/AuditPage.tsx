import { useEffect, useMemo, useState } from 'react'
import FilterAltRounded from '@mui/icons-material/FilterAltRounded'
import GavelRounded from '@mui/icons-material/GavelRounded'
import { Box, Button, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { auditApi } from '../api/audits'
import { PageHeader } from '../components/common/PageHeader'
import { ReviewDecisionDialog } from '../components/common/ReviewDecisionDialog'
import { useCaseStore } from '../stores/caseStore'
import { useObservationStore } from '../stores/observationStore'
import type { AuditEvent } from '../types/audit'
import type { CaseSummary } from '../types/case'
import { formatDateTime } from '../utils/format'

export function AuditPage() {
  const cases = useCaseStore((state) => state.cases)
  const loadCases = useCaseStore((state) => state.load)
  const transition = useCaseStore((state) => state.transition)
  const observations = useObservationStore((state) => state.observations)
  const loadObservations = useObservationStore((state) => state.load)
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [actorEmail, setActorEmail] = useState('')
  const [entityType, setEntityType] = useState('')
  const [reviewTarget, setReviewTarget] = useState<CaseSummary | null>(null)

  const loadAudits = async () => {
    const response = await auditApi.list({ actorEmail, entityType })
    setEvents(response.data)
  }

  useEffect(() => {
    void Promise.all([loadAudits(), loadCases(), loadObservations()])
  }, [])

  const caseById = useMemo(() => new Map(cases.map((item) => [item.id, item])), [cases])
  const observationById = useMemo(() => new Map(observations.map((item) => [item.id, item])), [observations])
  const pending = cases.find((item) => item.case_status === 'pending_review') ?? null

  const objectLabel = (event: AuditEvent) => {
    if (event.entity_type === 'interference_case') return caseById.get(event.entity_id)?.case_code ?? `案例 #${event.entity_id}`
    if (event.entity_type === 'bearing_observation') {
      const observation = observationById.get(event.entity_id)
      return observation ? `观测 #${observation.id} / 站点 ${observation.station?.station_code ?? observation.station_id}` : `观测 #${event.entity_id}`
    }
    return `${event.entity_type} #${event.entity_id}`
  }

  return (
    <>
      <PageHeader
        eyebrow="IMMUTABLE AUDIT / REQUEST TRACE"
        title="审计追踪"
        summary={`${events.length} 条不可变事件 · 参数、排除、定位与人工结论均保留 request ID`}
        actions={pending ? <Button variant="contained" startIcon={<GavelRounded />} onClick={() => setReviewTarget(pending)}>复核 {pending.case_code}</Button> : undefined}
      />

      <section className="control-strip">
        <TextField size="small" label="操作者邮箱" value={actorEmail} onChange={(event) => setActorEmail(event.target.value)} />
        <TextField size="small" label="对象类型" value={entityType} onChange={(event) => setEntityType(event.target.value)} placeholder="interference_case" />
        <Button variant="outlined" startIcon={<FilterAltRounded />} onClick={() => void loadAudits()}>应用筛选</Button>
      </section>

      <section className="data-section" aria-labelledby="audit-table-title">
        <Typography id="audit-table-title" component="h2" variant="h6" mb={2}>按时间倒序的不可变事件</Typography>
        <Box className="table-scroll">
          <Table size="small" aria-label="审计事件列表">
            <TableHead><TableRow><TableCell>时间 / 操作者</TableCell><TableCell>动作</TableCell><TableCell>对象</TableCell><TableCell>请求 ID</TableCell><TableCell>变更摘要</TableCell></TableRow></TableHead>
            <TableBody>{events.map((event) => <TableRow key={event.id} hover>
              <TableCell>{formatDateTime(event.created_at)}<br /><span className="secondary-text">{event.actor_email}</span></TableCell>
              <TableCell><code>{event.action}</code></TableCell>
              <TableCell><strong>{objectLabel(event)}</strong></TableCell>
              <TableCell><code className="request-id">{event.request_id}</code></TableCell>
              <TableCell><details><summary>查看前后摘要</summary><pre>{event.before_json}{'\n→\n'}{event.after_json}</pre></details></TableCell>
            </TableRow>)}</TableBody>
          </Table>
        </Box>
      </section>

      <ReviewDecisionDialog open={Boolean(reviewTarget)} item={reviewTarget} onClose={() => setReviewTarget(null)} onDecision={async (request) => { if (reviewTarget) { await transition(reviewTarget.id, request); await loadAudits() } }} />
    </>
  )
}

