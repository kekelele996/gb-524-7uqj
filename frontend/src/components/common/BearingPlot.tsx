import { useMemo } from 'react'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { LineChart, ScatterChart, type LineSeriesOption, type ScatterSeriesOption } from 'echarts/charts'
import { AriaComponent, GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption } from 'echarts/core'
import { Box, Typography } from '@mui/material'
import type { BearingObservation } from '../../types/observation'
import type { ReceiverStation } from '../../types/station'
import type { LocalizationEstimate } from '../../types/localization'
import { bearingVector, createFrame, toLocal, uncertaintyCircle } from '../../utils/geometry'
import { formatCoordinate, formatDecimal } from '../../utils/format'

interface BearingPlotProps {
  stations: ReceiverStation[]
  observations: BearingObservation[]
  estimate?: LocalizationEstimate | null
  height?: number
}

echarts.use([LineChart, ScatterChart, GridComponent, LegendComponent, TooltipComponent, AriaComponent, CanvasRenderer])

export function BearingPlot({ stations, observations, estimate, height = 440 }: BearingPlotProps) {
  const option = useMemo<EChartsCoreOption>(() => {
    const sourcePoints = stations.length > 0
      ? stations
      : estimate?.input_snapshot_json.map((item) => ({ latitude: item.latitude, longitude: item.longitude })) ?? []
    const frame = createFrame(sourcePoints)
    const localStations = stations.map((station) => ({ ...station, ...toLocal(frame, station.latitude, station.longitude) }))
    const scale = Math.max(3000, ...localStations.map((station) => Math.hypot(station.x, station.y) * 1.8))
    const series: Array<LineSeriesOption | ScatterSeriesOption> = []

    observations.filter((item) => item.quality !== 'excluded' && item.station).forEach((observation) => {
      const station = toLocal(frame, observation.station!.latitude, observation.station!.longitude)
      const vector = bearingVector(observation.corrected_bearing_deg, scale)
      series.push({
        name: `${observation.station!.station_code} ${observation.corrected_bearing_deg.toFixed(1)}°`,
        type: 'line',
        data: [[station.x, station.y], [station.x + vector.x, station.y + vector.y]],
        showSymbol: false,
        lineStyle: {
          color: observation.quality === 'poor' ? '#b86424' : observation.quality === 'fair' ? '#527c70' : '#1d5d50',
          width: observation.quality === 'good' ? 2 : 1.5,
          type: observation.quality === 'poor' ? 'dashed' : 'solid'
        }
      })
    })

    series.push({
      name: '测向站',
      type: 'scatter',
      symbol: 'triangle',
      symbolSize: 14,
      itemStyle: { color: '#173d35', borderColor: '#eef4ef', borderWidth: 2 },
      data: localStations.map((station) => ({ value: [station.x, station.y], name: station.station_code }))
    })

    if (estimate) {
      const center = toLocal(frame, estimate.latitude, estimate.longitude)
      series.push({
        name: '不确定范围',
        type: 'line',
        data: uncertaintyCircle(center, estimate.uncertainty_radius_m),
        showSymbol: false,
        lineStyle: { color: '#b86424', width: 1.5, type: 'dashed' },
        areaStyle: { color: 'rgba(184, 100, 36, 0.10)' }
      })
      series.push({
        name: estimate.estimate_status === 'outlier_candidate' ? '剔除候选点' : '加权估计点',
        type: 'scatter',
        symbol: 'diamond',
        symbolSize: 20,
        itemStyle: { color: '#b86424', borderColor: '#fff8ee', borderWidth: 3 },
        data: [[center.x, center.y]]
      })
    }

    return {
      animationDuration: 360,
      aria: { enabled: true, decal: { show: true }, description: '测向站、方位射线、定位估计点与不确定范围的本地坐标图' },
      backgroundColor: '#f8faf7',
      grid: { left: 64, right: 28, top: 52, bottom: 52 },
      legend: { top: 14, left: 18, type: 'scroll', textStyle: { color: '#315047', fontSize: 12 } },
      tooltip: { trigger: 'item', valueFormatter: (value: unknown) => `${Number(value).toFixed(1)} m` },
      xAxis: {
        type: 'value', name: '东向距离 / m', nameLocation: 'middle', nameGap: 32,
        splitLine: { lineStyle: { color: '#dce6df' } }, axisLine: { lineStyle: { color: '#789187' } }
      },
      yAxis: {
        type: 'value', name: '北向距离 / m', nameLocation: 'middle', nameGap: 46,
        splitLine: { lineStyle: { color: '#dce6df' } }, axisLine: { lineStyle: { color: '#789187' } },
        scale: true
      },
      series
    }
  }, [stations, observations, estimate])

  return (
    <Box component="figure" className="bearing-plot" aria-label="方位交汇坐标图">
      <ReactEChartsCore echarts={echarts} option={option} style={{ height }} notMerge />
      <Box component="figcaption" className="plot-evidence">
        {estimate ? (
          <>
            <Typography component="span"><strong>估计坐标</strong> {formatCoordinate(estimate.latitude)}, {formatCoordinate(estimate.longitude)}</Typography>
            <Typography component="span"><strong>残差</strong> {formatDecimal(estimate.residual_deg)}°</Typography>
            <Typography component="span"><strong>不确定半径</strong> {formatDecimal(estimate.uncertainty_radius_m, 0)} m</Typography>
            <Typography component="span"><strong>条件数</strong> {formatDecimal(estimate.condition_number)}</Typography>
          </>
        ) : (
          <Typography component="span">当前显示 {stations.length} 个测向站与 {observations.filter((item) => item.quality !== 'excluded').length} 条有效方位线。</Typography>
        )}
      </Box>
    </Box>
  )
}
