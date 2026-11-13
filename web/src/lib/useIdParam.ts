import { useParams } from 'react-router'

// useIdParam — числовой :id из адреса; null, если в адресе мусор
export function useIdParam(): number | null {
  const { id } = useParams()
  const n = Number(id)
  return Number.isInteger(n) && n > 0 ? n : null
}
