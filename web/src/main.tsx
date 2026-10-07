import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { App, Dashboard, FrameworkPage, SettingsPage } from './app/App'
import './styles.css'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 10_000 } } })

createRoot(document.getElementById('root')!).render(<StrictMode><QueryClientProvider client={queryClient}><BrowserRouter><Routes><Route element={<App />}><Route index element={<Dashboard />} /><Route path="framework" element={<FrameworkPage />} /><Route path="settings" element={<SettingsPage />} /></Route></Routes></BrowserRouter></QueryClientProvider></StrictMode>)
