import { workspaceQueryKey } from './query'
import { describe, expect, it } from 'vitest'

describe('workspaceQueryKey', () => {
  it('keeps every workspace cache isolated', () => {
    expect(workspaceQueryKey('workspace-a', 'members')).toEqual(['workspace', 'workspace-a', 'members'])
    expect(workspaceQueryKey('workspace-b', 'members')).not.toEqual(workspaceQueryKey('workspace-a', 'members'))
  })
})
