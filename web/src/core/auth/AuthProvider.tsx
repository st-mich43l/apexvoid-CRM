import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { CurrentUser } from '../../framework/auth/types'
import { api } from '../api/client'
import { AuthContext, type AuthContextValue } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [platformPermissions, setPlatformPermissions] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const identityRef = useRef<string | null>(null)

  const clearIdentityCache = useCallback(async () => {
    await queryClient.cancelQueries()
    queryClient.clear()
  }, [queryClient])

  const applyAuth = useCallback(async (response: { user: CurrentUser; permissions?: string[] }) => {
    if (identityRef.current !== null && identityRef.current !== response.user.id) await clearIdentityCache()
    identityRef.current = response.user.id
    setUser(response.user)
    setPlatformPermissions(response.permissions ?? [])
  }, [clearIdentityCache])

  const reload = useCallback(async () => {
    try {
      const response = await api.auth.me()
      await applyAuth(response)
    } catch (error) {
      if ((error as { status?: number }).status === 401) {
        try { await api.auth.refresh(); const response = await api.auth.me(); await applyAuth(response) } catch { await clearIdentityCache(); identityRef.current = null; setUser(null); setPlatformPermissions([]) }
      } else { await clearIdentityCache(); identityRef.current = null; setUser(null); setPlatformPermissions([]) }
    }
  }, [applyAuth, clearIdentityCache])

  useEffect(() => { void reload().finally(() => setLoading(false)) }, [reload])
  const value = useMemo<AuthContextValue>(() => ({ user, loading, platformPermissions, login: async (email, password) => { const response = await api.auth.login(email, password); await clearIdentityCache(); await applyAuth(response) }, logout: async () => { try { await api.auth.logout() } finally { await clearIdentityCache(); identityRef.current = null; window.localStorage.removeItem('apexvoid.active_workspace'); setUser(null); setPlatformPermissions([]) } }, platformCan: permission => platformPermissions.includes(permission), reload }), [applyAuth, clearIdentityCache, user, loading, platformPermissions, reload])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
