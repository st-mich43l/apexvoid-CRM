import { userWorkspaceQueryKey, workspaceQueryKey } from './query'
import { describe, expect, it } from 'vitest'

describe('workspaceQueryKey', () => {
  it('keeps every workspace cache isolated', () => {
    expect(workspaceQueryKey('workspace-a', 'members')).toEqual(['workspace', 'workspace-a', 'members'])
    expect(workspaceQueryKey('workspace-b', 'members')).not.toEqual(workspaceQueryKey('workspace-a', 'members'))
  })
})

it('isolates authorization-sensitive workspace data by identity', () => {
  expect(userWorkspaceQueryKey('user-a', 'workspace-a', 'members')).not.toEqual(userWorkspaceQueryKey('user-b', 'workspace-a', 'members'))
  expect(userWorkspaceQueryKey('user-a', 'workspace-a', 'members')).not.toEqual(userWorkspaceQueryKey('user-a', 'workspace-b', 'members'))
})
