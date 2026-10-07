import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { CurrentUser } from '../../framework/auth/types'
import { api } from '../api/client'
import { AuthContext, type AuthContextValue } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [permissions, setPermissions] = useState<string[]>([])
  const [loading, setLoading] = useState(true)

  const reload = useCallback(async () => {
    try {
      const response = await api.auth.me()
      setUser(response.user)
      setPermissions(response.permissions ?? [])
    } catch (error) {
      if ((error as { status?: number }).status === 401) {
        try { await api.auth.refresh(); const response = await api.auth.me(); setUser(response.user); setPermissions(response.permissions ?? []) } catch { setUser(null); setPermissions([]) }
      } else { setUser(null); setPermissions([]) }
    }
  }, [])

  useEffect(() => { void reload().finally(() => setLoading(false)) }, [reload])
  const value = useMemo<AuthContextValue>(() => ({ user, loading, permissions, login: async (email, password) => { const response = await api.auth.login(email, password); setUser(response.user); setPermissions(response.permissions ?? []) }, logout: async () => { try { await api.auth.logout() } finally { window.localStorage.removeItem('apexvoid.active_workspace'); setUser(null); setPermissions([]) } }, can: permission => permissions.includes(permission), reload }), [user, loading, permissions, reload])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
