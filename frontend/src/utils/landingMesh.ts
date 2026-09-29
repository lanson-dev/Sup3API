type Vector = [number, number, number]
interface Mesh {
  vertices: Vector[]
  faces: [number, number, number][]
}
const normalize = (v: Vector): Vector => {
  const length = Math.hypot(...v) || 1
  return v.map((n) => n / length) as Vector
}
const subtract = (a: Vector, b: Vector): Vector => [
  a[0] - b[0],
  a[1] - b[1],
  a[2] - b[2],
]
const cross = (a: Vector, b: Vector): Vector => [
  a[1] * b[2] - a[2] * b[1],
  a[2] * b[0] - a[0] * b[2],
  a[0] * b[1] - a[1] * b[0],
]
const center = (t: number): Vector => [
  (2 + 0.7 * Math.cos(3 * t)) * Math.cos(2 * t),
  (2 + 0.7 * Math.cos(3 * t)) * Math.sin(2 * t),
  0.85 * Math.sin(3 * t),
]

/** Local decorative geometry; never fetches or generates a provider asset. */
export function createKnotMesh(): Mesh {
  const rings = 96,
    sides = 12
  const vertices: Vector[] = []
  const faces: Mesh['faces'] = []
  for (let ring = 0; ring < rings; ring++) {
    const t = (ring / rings) * Math.PI * 2
    const point = center(t)
    const tangent = normalize(subtract(center(t + 0.001), center(t - 0.001)))
    const normal = normalize(cross(tangent, [0, 0, 1]))
    const binormal = normalize(cross(tangent, normal))
    for (let side = 0; side < sides; side++) {
      const v = (side / sides) * Math.PI * 2
      vertices.push(
        point.map(
          (p, axis) =>
            p +
            0.48 * (Math.cos(v) * normal[axis] + Math.sin(v) * binormal[axis]),
        ) as Vector,
      )
      const a = ring * sides + side,
        b = ((ring + 1) % rings) * sides + side
      const c = ((ring + 1) % rings) * sides + ((side + 1) % sides),
        d = ring * sides + ((side + 1) % sides)
      faces.push([a, b, c], [a, c, d])
    }
  }
  return { vertices, faces }
}

export function paintKnot(
  ctx: CanvasRenderingContext2D,
  mesh: Mesh,
  width: number,
  height: number,
  rotation: { x: number; y: number },
  wireframe: boolean,
) {
  ctx.clearRect(0, 0, width, height)
  const scale = Math.min(width, height) / 7.8
  const sinX = Math.sin(rotation.x),
    cosX = Math.cos(rotation.x),
    sinY = Math.sin(rotation.y),
    cosY = Math.cos(rotation.y)
  const vertices = mesh.vertices.map(([x, y, z]): Vector => {
    const ry = y * cosX - z * sinX,
      rz = y * sinX + z * cosX
    return [x * cosY + rz * sinY, ry, -x * sinY + rz * cosY]
  })
  const projected = vertices.map(([x, y, z]) => {
    const p = 10 / (10 - z)
    return [width / 2 + x * scale * p, height / 2 + y * scale * p]
  })
  const sorted = mesh.faces
    .map((face) => ({
      face,
      z: face.reduce((sum, index) => sum + vertices[index][2], 0),
    }))
    .sort((a, b) => a.z - b.z)
  ctx.lineJoin = 'round'
  for (const {
    face: [a, b, c],
  } of sorted) {
    const normal = normalize(
      cross(
        subtract(vertices[c], vertices[a]),
        subtract(vertices[b], vertices[a]),
      ),
    )
    // Cull surfaces facing away from the perspective camera before canvas calls.
    const [x, y, z] = vertices[a]
    if (-x * normal[0] - y * normal[1] + (10 - z) * normal[2] <= 0) continue
    const light = Math.max(
      0,
      normal[0] * -0.45 + normal[1] * -0.65 + normal[2] * 0.61,
    )
    const specular = Math.pow(
      Math.max(0, normal[0] * -0.15 + normal[1] * -0.38 + normal[2] * 0.91),
      22,
    )
    const shade = 18 + light * 48 + specular * 28
    ctx.beginPath()
    ctx.moveTo(projected[a][0], projected[a][1])
    ctx.lineTo(projected[b][0], projected[b][1])
    ctx.lineTo(projected[c][0], projected[c][1])
    ctx.closePath()
    ctx.fillStyle = wireframe
      ? '#0c1915'
      : `hsl(${142 + light * 12} ${25 + light * 17}% ${shade}%)`
    ctx.fill()
    ctx.strokeStyle = wireframe
      ? `rgba(166, 232, 185, ${0.24 + light * 0.48})`
      : ctx.fillStyle
    ctx.lineWidth = wireframe ? 0.65 : 0.5
    ctx.stroke()
  }
}
