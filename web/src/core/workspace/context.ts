import { createContext, useContext } from 'react'
import type { SetupStatus, Workspace } from '../api/client'

export type WorkspaceContextValue = {
  workspaces: Workspace[]
  activeWorkspace: Workspace | null
  activeWorkspaceId: string | null
  setup: SetupStatus | null
  loading: boolean
  selectWorkspace: (id: string) => void
  reload: () => Promise<void>
}

export const WorkspaceContext = createContext<WorkspaceContextValue | null>(null)

export function useWorkspace() {
  const context = useContext(WorkspaceContext)
  if (!context) throw new Error('useWorkspace must be used inside WorkspaceProvider')
  return context
}
