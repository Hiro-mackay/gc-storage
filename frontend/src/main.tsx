import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { router } from '@/app/router'
import { api } from '@/lib/api/client'
import { useAuthStore } from '@/stores/auth-store'
import './index.css'

// Initialize auth state by checking session
async function initAuth() {
  try {
    const { data } = await api.GET('/me')
    if (data?.data) {
      useAuthStore.getState().setUser(data.data)
    } else {
      useAuthStore.getState().clearAuth()
    }
  } catch {
    useAuthStore.getState().clearAuth()
  }
}

initAuth().then(() => {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <RouterProvider router={router} />
    </StrictMode>
  )
})
