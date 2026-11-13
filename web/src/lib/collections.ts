export const indexById = <T extends { id: number }>(items: T[]) =>
  Object.fromEntries(items.map((item) => [item.id, item])) as Record<number, T>
