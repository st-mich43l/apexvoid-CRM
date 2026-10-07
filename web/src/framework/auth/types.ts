export type CurrentUser = { id: string; email: string; username?: string; display_name: string; status: string }
export type AuthResponse = { user: CurrentUser; permissions: string[] }
export type Permission = { name: string; module: string; display_name: string; description: string }
