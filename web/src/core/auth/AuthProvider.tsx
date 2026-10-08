import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { CurrentUser } from '../../framework/auth/types'
import { api } from '../api/client'
import { AuthContext, type AuthContextValue } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [platformPermissions, setPlatformPermissions] = useState<string[]>([])
  const [loading, setLoading] = useState(true)

  const reload = useCallback(async () => {
    try {
      const response = await api.auth.me()
      setUser(response.user)
      setPlatformPermissions(response.permissions ?? [])
    } catch (error) {
      if ((error as { status?: number }).status === 401) {
        try { await api.auth.refresh(); const response = await api.auth.me(); setUser(response.user); setPlatformPermissions(response.permissions ?? []) } catch { setUser(null); setPlatformPermissions([]) }
      } else { setUser(null); setPlatformPermissions([]) }
    }
  }, [])

  useEffect(() => { void reload().finally(() => setLoading(false)) }, [reload])
  const value = useMemo<AuthContextValue>(() => ({ user, loading, platformPermissions, login: async (email, password) => { const response = await api.auth.login(email, password); setUser(response.user); setPlatformPermissions(response.permissions ?? []) }, logout: async () => { try { await api.auth.logout() } finally { window.localStorage.removeItem('apexvoid.active_workspace'); setUser(null); setPlatformPermissions([]) } }, platformCan: permission => platformPermissions.includes(permission), reload }), [user, loading, platformPermissions, reload])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
