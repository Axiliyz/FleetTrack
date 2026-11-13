import { useSearchParams } from 'react-router'

// useListParams держит фильтры и страницу списка в адресной строке:
// ссылку на отфильтрованный список можно отправить коллеге, а «назад» в браузере возвращает фильтр
export function useListParams<K extends string>(keys: readonly K[]) {
  const [params, setParams] = useSearchParams()

  const filter = Object.fromEntries(keys.map((key) => [key, params.get(key) ?? ''])) as Record<K, string>
  const offset = Math.max(0, Number(params.get('offset')) || 0)

  const update = (patch: Record<string, string>) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        for (const [key, value] of Object.entries(patch)) {
          if (value) next.set(key, value)
          else next.delete(key)
        }
        return next
      },
      { replace: true },
    )

  return {
    filter,
    offset,
    active: keys.some((key) => filter[key] !== ''),
    // новый фильтр — снова с первой страницы
    set: (key: K, value: string) => update({ [key]: value, offset: '' }),
    setOffset: (next: number) => update({ offset: next > 0 ? String(next) : '' }),
    reset: () => update(Object.fromEntries([...keys, 'offset'].map((key) => [key, '']))),
  }
}
