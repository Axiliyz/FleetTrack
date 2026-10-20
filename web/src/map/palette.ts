// Цвета для deck.gl. Те же значения заданы CSS-переменными в styles.css — меняйте парой.
import type { Condition } from '../state/selectors'

export type RGB = [number, number, number]

export const LAMP_RGB: Record<Condition, RGB> = {
  trip: [242, 163, 58],
  idle: [107, 213, 143],
  service: [116, 169, 255],
  alarm: [255, 82, 67],
  offline: [86, 99, 92],
}

export const INK_RGB: RGB = [214, 222, 207]
export const BOARD_RGB: RGB = [14, 22, 19]
