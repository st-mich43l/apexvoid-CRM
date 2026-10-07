import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'
import { appModules } from './app/bootstrap/modules'
import { createAppRouter } from './app/router/router'
import { AuthProvider } from './core/auth/AuthProvider'
import { ThemeProvider } from './core/theme/ThemeProvider'
import './styles.css'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 10_000 } } })
const router = createAppRouter(appModules)

createRoot(document.getElementById('root')!).render(<StrictMode><QueryClientProvider client={queryClient}><ThemeProvider><AuthProvider><RouterProvider router={router} /></AuthProvider></ThemeProvider></QueryClientProvider></StrictMode>)
