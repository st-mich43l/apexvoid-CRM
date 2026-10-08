import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { WorkspaceContext, type WorkspaceContextValue } from './context'
import { userWorkspaceQueryKey } from './query'
import { useAuth } from '../auth/context'

const storageKey = 'apexvoid.active_workspace'

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(() => window.localStorage.getItem(storageKey))
  const [pendingWorkspaceId, setPendingWorkspaceId] = useState<string | null>(null)
  const [ownerID, setOwnerID] = useState<string | undefined>(user?.id)
  const effectiveWorkspaceId = ownerID === user?.id ? activeWorkspaceId : null
  const workspaces = useQuery({ queryKey: ['workspaces', user?.id ?? ''], queryFn: api.workspaces.list, enabled: Boolean(user), retry: false })
  const setup = useQuery({ queryKey: ['setup-status', user?.id ?? ''], queryFn: api.setup.status, enabled: Boolean(user), retry: false })
  const current = useQuery({ queryKey: userWorkspaceQueryKey(user?.id, effectiveWorkspaceId, 'context'), queryFn: api.workspaces.current, enabled: Boolean(user && effectiveWorkspaceId), retry: false })

  useEffect(() => {
    setOwnerID(user?.id)
    setActiveWorkspaceId(null)
    setPendingWorkspaceId(null)
    window.localStorage.removeItem(storageKey)
  }, [user?.id])

  useEffect(() => {
    if (!workspaces.data) return
    if (pendingWorkspaceId) {
      if (workspaces.data.some(item => item.id === pendingWorkspaceId)) {
        setActiveWorkspaceId(pendingWorkspaceId)
        window.localStorage.setItem(storageKey, pendingWorkspaceId)
        setPendingWorkspaceId(null)
        return
      }
      if (workspaces.isFetching) return
    }
    const remembered = activeWorkspaceId && workspaces.data.some(item => item.id === activeWorkspaceId) ? activeWorkspaceId : workspaces.data[0]?.id ?? null
    setActiveWorkspaceId(remembered)
    if (remembered) window.localStorage.setItem(storageKey, remembered)
    else window.localStorage.removeItem(storageKey)
  }, [activeWorkspaceId, pendingWorkspaceId, workspaces.data, workspaces.isFetching])

  const selectWorkspace = useCallback((id: string) => {
    window.localStorage.setItem(storageKey, id)
    setPendingWorkspaceId(id)
    setActiveWorkspaceId(id)
    void queryClient.invalidateQueries({ queryKey: ['workspace', user?.id ?? ''] })
  }, [queryClient, user?.id])

  const reload = useCallback(async () => {
    await Promise.all([queryClient.invalidateQueries({ queryKey: ['workspaces', user?.id ?? ''] }), queryClient.invalidateQueries({ queryKey: ['setup-status', user?.id ?? ''] })])
  }, [queryClient, user?.id])

  const workspacePermissions = useMemo(() => current.isSuccess ? current.data?.permissions ?? [] : [], [current.data?.permissions, current.isSuccess])
  const value = useMemo<WorkspaceContextValue>(() => ({ workspaces: workspaces.data ?? [], activeWorkspace: workspaces.data?.find(item => item.id === activeWorkspaceId) ?? null, activeWorkspaceId, setup: setup.data ?? null, workspacePermissions, loading: workspaces.isLoading || setup.isLoading || current.isLoading, selectWorkspace, can: permission => workspacePermissions.includes(permission), reload }), [activeWorkspaceId, current.isLoading, reload, selectWorkspace, setup.data, setup.isLoading, workspacePermissions, workspaces.data, workspaces.isLoading])
  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>
}
