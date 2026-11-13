import { setWorkerUrl } from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import maplibreWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url'

// тёмная подложка без подписей: города подписываем сами, по-русски
export const STYLE_URL = 'https://basemaps.cartocdn.com/gl/dark-matter-nolabels-gl-style/style.json'

// maplibre ищет воркер рядом со своим модулем, а Vite при сборке зависимостей переносит модуль —
// поэтому путь к воркеру задаём явно
setWorkerUrl(maplibreWorkerUrl)
