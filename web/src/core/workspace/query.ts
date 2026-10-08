export function workspaceQueryKey(workspaceID: string | null | undefined, ...parts: string[]) {
  return ['workspace', workspaceID ?? '', ...parts] as const
}
