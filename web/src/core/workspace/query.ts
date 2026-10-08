export function workspaceQueryKey(workspaceID: string | null | undefined, ...parts: string[]) {
  return ['workspace', workspaceID ?? '', ...parts] as const
}

export function userWorkspaceQueryKey(userID: string | null | undefined, workspaceID: string | null | undefined, ...parts: string[]) {
  return ['workspace', userID ?? '', workspaceID ?? '', ...parts] as const
}

export function userQueryKey(userID: string | null | undefined, ...parts: string[]) {
  return ['user', userID ?? '', ...parts] as const
}
