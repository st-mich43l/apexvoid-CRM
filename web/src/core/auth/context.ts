import { createContext, useContext } from 'react'
import type { CurrentUser } from '../../framework/auth/types'

export type AuthContextValue = { user: CurrentUser | null; loading: boolean; platformPermissions: string[]; login: (email: string, password: string) => Promise<void>; logout: () => Promise<void>; platformCan: (permission: string) => boolean; reload: () => Promise<void> }
export const AuthContext = createContext<AuthContextValue | null>(null)
export function useAuth() { const context = useContext(AuthContext); if (!context) throw new Error('useAuth must be used inside AuthProvider'); return context }
