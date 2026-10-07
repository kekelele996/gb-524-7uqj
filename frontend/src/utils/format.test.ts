import { describe, expect, it } from 'vitest'
import { formatFrequency } from './format'
import { bearingVector, createFrame, toLocal } from './geometry'

describe('formatFrequency', () => {
  it('formats radio frequencies with engineering units', () => {
    expect(formatFrequency(433_920_000)).toBe('433.920000 MHz')
  })
})

describe('local coordinate helpers', () => {
  it('keeps the frame origin at zero', () => {
    const frame = createFrame([{ latitude: 31.23, longitude: 121.47 }])
    expect(toLocal(frame, 31.23, 121.47)).toEqual({ x: 0, y: 0 })
  })

  it('uses north-up bearing convention', () => {
    const north = bearingVector(0, 100)
    expect(north.x).toBeCloseTo(0, 8)
    expect(north.y).toBeCloseTo(100, 8)
  })
})

