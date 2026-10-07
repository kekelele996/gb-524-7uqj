const EARTH_RADIUS_M = 6_371_008.8

export interface LocalPoint {
  x: number
  y: number
}

export interface CoordinateFrame {
  latitude: number
  longitude: number
  cosLatitude: number
}

export function createFrame(points: Array<{ latitude: number; longitude: number }>): CoordinateFrame {
  const count = Math.max(points.length, 1)
  const latitude = points.reduce((sum, point) => sum + point.latitude, 0) / count
  const longitude = points.reduce((sum, point) => sum + point.longitude, 0) / count
  return { latitude, longitude, cosLatitude: Math.cos(toRadians(latitude)) }
}

export function toLocal(frame: CoordinateFrame, latitude: number, longitude: number): LocalPoint {
  return {
    x: EARTH_RADIUS_M * toRadians(longitude - frame.longitude) * frame.cosLatitude,
    y: EARTH_RADIUS_M * toRadians(latitude - frame.latitude)
  }
}

export function bearingVector(bearing: number, length: number): LocalPoint {
  const radians = toRadians(bearing)
  return { x: Math.sin(radians) * length, y: Math.cos(radians) * length }
}

export function uncertaintyCircle(center: LocalPoint, radius: number): Array<[number, number]> {
  return Array.from({ length: 73 }, (_, index) => {
    const angle = (index / 72) * Math.PI * 2
    return [center.x + Math.cos(angle) * radius, center.y + Math.sin(angle) * radius]
  })
}

function toRadians(value: number): number {
  return value * Math.PI / 180
}

