import { create } from 'zustand'
import type { components } from '@/lib/api/schema'

type User = components['schemas']['github_com_Hiro-mackay_gc-storage_backend_internal_interface_dto_response.UserResponse']

type AuthStatus = 'initializing' | 'authenticated' | 'unauthenticated'

interface AuthState {
  status: AuthStatus
  user: User | null
  setUser: (user: User) => void
  clearAuth: () => void
  setInitializing: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  status: 'initializing',
  user: null,
  setUser: (user) => set({ status: 'authenticated', user }),
  clearAuth: () => set({ status: 'unauthenticated', user: null }),
  setInitializing: () => set({ status: 'initializing' }),
}))
