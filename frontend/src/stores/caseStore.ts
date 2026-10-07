import { create } from 'zustand'
import { caseApi } from '../api/cases'
import type { CaseInput, CaseSummary, CaseTransition } from '../types/case'

interface CaseState {
  cases: CaseSummary[]
  busy: boolean
  load: () => Promise<void>
  createCase: (input: CaseInput) => Promise<void>
  transition: (id: number, input: CaseTransition) => Promise<void>
}

export const useCaseStore = create<CaseState>((set, get) => ({
  cases: [],
  busy: false,
  load: async () => {
    set({ busy: true })
    try {
      const response = await caseApi.list()
      set({ cases: response.data })
    } finally {
      set({ busy: false })
    }
  },
  createCase: async (input) => {
    await caseApi.create(input)
    await get().load()
  },
  transition: async (id, input) => {
    await caseApi.transition(id, input)
    await get().load()
  }
}))

