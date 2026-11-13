import type { LayersList } from '@deck.gl/core'
import { LineLayer, PathLayer, ScatterplotLayer, TextLayer } from '@deck.gl/layers'
import type { LngLat } from '../lib/geo'
import type { Condition } from '../state/selectors'
import { motion, type TrailSegment } from './motion'
import { CORRIDORS, MKAD_RING, PLACES, corridorPath, type Place } from './network'
import { BOARD_RGB, INK_RGB, LAMP_RGB } from './palette'

// с этого зуума показываем все города и номера машин
export const NEAR_ZOOM = 7

const FONT = '"IBM Plex Sans Condensed", "Arial Narrow", sans-serif'

interface PathDatum {
  path: LngLat[]
}

interface CodeLabel {
  code: string
  pos: LngLat
}

const CORRIDOR_PATHS: PathDatum[] = CORRIDORS.map((c) => ({ path: corridorPath(c) }))
const RING: PathDatum[] = [{ path: MKAD_RING }]
const MAJOR_PLACES = PLACES.filter((p) => p.major)

// номер трассы ставим между вторым и третьим городом — там, где линии уже разошлись
const CODE_LABELS: CodeLabel[] = CORRIDORS.map((c) => {
  const [a, b] = [c.places[1].pos, c.places[2].pos]
  return { code: c.code, pos: [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2] }
})

export function buildLayers(opts: {
  zoom: number
  fontsReady: boolean
  conditionOf: (vehicleId: number) => Condition
}): LayersList {
  const near = opts.zoom >= NEAR_ZOOM
  const ink = (alpha: number): [number, number, number, number] => [...INK_RGB, alpha]

  const layers: LayersList = [
    new PathLayer<PathDatum>({
      id: 'corridor-glow',
      data: CORRIDOR_PATHS,
      getPath: (d) => d.path,
      getColor: ink(18),
      getWidth: 10,
      widthUnits: 'pixels',
      jointRounded: true,
      capRounded: true,
    }),
    new PathLayer<PathDatum>({
      id: 'corridor',
      data: CORRIDOR_PATHS,
      getPath: (d) => d.path,
      getColor: ink(95),
      getWidth: 1.6,
      widthUnits: 'pixels',
      jointRounded: true,
    }),
    new PathLayer<PathDatum>({
      id: 'mkad',
      data: RING,
      getPath: (d) => d.path,
      getColor: ink(130),
      getWidth: 2,
      widthUnits: 'pixels',
    }),
    new LineLayer<TrailSegment>({
      id: 'trails',
      data: motion.trailSegments(),
      getSourcePosition: (d) => d.from,
      getTargetPosition: (d) => d.to,
      getColor: (d) => [...LAMP_RGB[opts.conditionOf(d.id)], Math.round(40 + d.strength * 190)],
      getWidth: (d) => 0.8 + d.strength * 3.2,
      widthUnits: 'pixels',
    }),
    new ScatterplotLayer<Place>({
      id: 'places',
      data: PLACES,
      getPosition: (d) => d.pos,
      getRadius: (d) => (d.major ? 4.5 : 2.5),
      radiusUnits: 'pixels',
      filled: true,
      getFillColor: [...BOARD_RGB, 255],
      stroked: true,
      getLineColor: ink(210),
      getLineWidth: 1.5,
      lineWidthUnits: 'pixels',
    }),
  ]

  // атлас символов строится один раз: ждём шрифт, иначе подписи останутся системным шрифтом
  if (opts.fontsReady) {
    layers.push(
      new TextLayer<Place>({
        id: 'place-labels',
        data: near ? PLACES : MAJOR_PLACES,
        getPosition: (d) => d.pos,
        getText: (d) => d.name,
        getSize: (d) => (d.major ? 13 : 11),
        getColor: (d) => ink(d.major ? 225 : 150),
        getTextAnchor: 'start',
        getAlignmentBaseline: 'center',
        getPixelOffset: [9, 0],
        fontFamily: FONT,
        fontWeight: 500,
        characterSet: 'auto',
        fontSettings: { sdf: true },
        outlineWidth: 4,
        outlineColor: [...BOARD_RGB, 255],
      }),
      new TextLayer<CodeLabel>({
        id: 'corridor-codes',
        data: CODE_LABELS,
        getPosition: (d) => d.pos,
        getText: (d) => d.code,
        getSize: 11,
        getColor: ink(230),
        fontFamily: FONT,
        fontWeight: 600,
        characterSet: 'auto',
        background: true,
        getBackgroundColor: [...BOARD_RGB, 255],
        getBorderColor: ink(160),
        getBorderWidth: 1,
        backgroundPadding: [4, 2],
      }),
    )
  }

  return layers
}
