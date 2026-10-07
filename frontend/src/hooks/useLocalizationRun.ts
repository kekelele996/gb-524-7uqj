import { useCallback, useState } from 'react'
import { useLocalizationStore } from '../stores/localizationStore'
import type { LocalizationRunResult } from '../types/localization'

export function useLocalizationRun() {
  const run = useLocalizationStore((state) => state.run)
  const busy = useLocalizationStore((state) => state.busy)
  const [lastResult, setLastResult] = useState<LocalizationRunResult | null>(null)

  const execute = useCallback(async (caseId: number, allowOutlier = true) => {
    const result = await run(caseId, allowOutlier)
    setLastResult(result)
    return result
  }, [run])

  return { execute, busy, lastResult }
}

