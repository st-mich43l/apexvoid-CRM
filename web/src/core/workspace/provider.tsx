import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { WorkspaceContext, type WorkspaceContextValue } from './context'

const storageKey = 'apexvoid.active_workspace'

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(() => window.localStorage.getItem(storageKey))
  const workspaces = useQuery({ queryKey: ['workspaces'], queryFn: api.workspaces.list, retry: false })
  const setup = useQuery({ queryKey: ['setup-status'], queryFn: api.setup.status, retry: false })

  useEffect(() => {
    if (!workspaces.data) return
    const remembered = activeWorkspaceId && workspaces.data.some(item => item.id === activeWorkspaceId) ? activeWorkspaceId : workspaces.data[0]?.id ?? null
    setActiveWorkspaceId(remembered)
    if (remembered) window.localStorage.setItem(storageKey, remembered)
    else window.localStorage.removeItem(storageKey)
  }, [workspaces.data, activeWorkspaceId])

  const selectWorkspace = useCallback((id: string) => {
    if (!workspaces.data?.some(item => item.id === id)) return
    window.localStorage.setItem(storageKey, id)
    setActiveWorkspaceId(id)
    void queryClient.invalidateQueries({ predicate: query => query.queryKey[0] === 'workspace' || query.queryKey[0] === 'members' || query.queryKey[0] === 'roles' })
  }, [queryClient, workspaces.data])

  const reload = useCallback(async () => {
    await Promise.all([queryClient.invalidateQueries({ queryKey: ['workspaces'] }), queryClient.invalidateQueries({ queryKey: ['setup-status'] })])
  }, [queryClient])

  const value = useMemo<WorkspaceContextValue>(() => ({ workspaces: workspaces.data ?? [], activeWorkspace: workspaces.data?.find(item => item.id === activeWorkspaceId) ?? null, activeWorkspaceId, setup: setup.data ?? null, loading: workspaces.isLoading || setup.isLoading, selectWorkspace, reload }), [activeWorkspaceId, reload, selectWorkspace, setup.data, setup.isLoading, workspaces.data, workspaces.isLoading])
  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>
}
