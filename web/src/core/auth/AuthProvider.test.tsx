import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from './AuthProvider'
import { useAuth } from './context'

const authAPI = vi.hoisted(() => ({
  me: vi.fn(),
  refresh: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('../api/client', () => ({ api: { auth: authAPI } }))

const userA = { id: 'user-a', email: 'a@example.test', display_name: 'User A', status: 'active', must_change_password: false }
const userB = { id: 'user-b', email: 'b@example.test', display_name: 'User B', status: 'active', must_change_password: false }

function Probe() {
  const { user, login } = useAuth()
  const client = useQueryClient()
  return <><span>{user?.display_name}</span><button onClick={() => void login('b@example.test', 'password')}>Login B</button><button onClick={() => client.setQueryData(['user', userA.id, 'members'], ['private A data'])}>Seed A</button></>
}

describe('AuthProvider cache isolation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authAPI.me.mockResolvedValue({ user: userA, permissions: ['workspace.member.read'] })
    authAPI.login.mockResolvedValue({ user: userB, permissions: [] })
  })

  it('restores a first-party session after the access cookie expires', async () => {
    authAPI.me.mockReset()
    authAPI.me.mockRejectedValueOnce(Object.assign(new Error('Access expired'), { status: 401 }))
      .mockResolvedValue({ user: userA, permissions: [] })
    authAPI.refresh.mockResolvedValue({ user: userA, permissions: [] })
    const client = new QueryClient()
    render(<QueryClientProvider client={client}><AuthProvider><Probe /></AuthProvider></QueryClientProvider>)
    await waitFor(() => expect(screen.getByText('User A')).toBeInTheDocument())
    expect(authAPI.refresh).toHaveBeenCalledTimes(1)
    expect(authAPI.me).toHaveBeenCalledTimes(2)
  })

  it('removes the previous identity cache when an account changes', async () => {
    const client = new QueryClient()
    render(<QueryClientProvider client={client}><AuthProvider><Probe /></AuthProvider></QueryClientProvider>)
    await waitFor(() => expect(screen.getByText('User A')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: 'Seed A' }))
    expect(client.getQueryData(['user', userA.id, 'members'])).toEqual(['private A data'])
    fireEvent.click(screen.getByRole('button', { name: 'Login B' }))
    await waitFor(() => expect(screen.getByText('User B')).toBeInTheDocument())
    expect(client.getQueryData(['user', userA.id, 'members'])).toBeUndefined()
  })
})
