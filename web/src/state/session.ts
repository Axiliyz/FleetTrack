import { useSyncExternalStore } from 'react'
import { auth } from '../api/client'
import { LIVE } from '../config'
import { accessOf, FULL_ACCESS, type Access } from '../lib/access'
import type { Claims } from '../types'

export const useSession = () => useSyncExternalStore(auth.onChange, auth.current)

export const useClaims = (): Claims | null => useSyncExternalStore(auth.onChange, auth.claims)

export function useAccess(): Access {
  const claims = useClaims()
  return LIVE ? accessOf(claims?.role) : FULL_ACCESS
}
