import {
  createRouter,
  createRootRoute,
  createRoute,
  redirect,
} from '@tanstack/react-router'
import { QueryClient } from '@tanstack/react-query'
import { RootLayout } from '@/components/layout/root-layout'
import { AuthLayout } from '@/components/layout/auth-layout'
import { MainLayout } from '@/components/layout/main-layout'
import { LoginPage } from '@/features/auth/pages/login-page'
import { RegisterPage } from '@/features/auth/pages/register-page'
import { FileBrowserPage } from '@/features/files/pages/file-browser-page'
import { TrashPage } from '@/features/trash/pages/trash-page'
import { SettingsPage } from '@/features/settings/pages/settings-page'
import { GroupsPage } from '@/features/groups/pages/groups-page'
import { GroupDetailPage } from '@/features/groups/pages/group-detail-page'
import { useAuthStore } from '@/stores/auth-store'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      retry: 1,
    },
  },
})

// Root route
const rootRoute = createRootRoute({
  component: RootLayout,
})

// Auth layout route (public - login, register)
const authLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'auth',
  component: AuthLayout,
  beforeLoad: () => {
    const { status } = useAuthStore.getState()
    if (status === 'authenticated') {
      throw redirect({ to: '/files' })
    }
  },
})

const loginRoute = createRoute({
  getParentRoute: () => authLayoutRoute,
  path: '/login',
  component: LoginPage,
})

const registerRoute = createRoute({
  getParentRoute: () => authLayoutRoute,
  path: '/register',
  component: RegisterPage,
})

// Authenticated layout route
const authenticatedLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'authenticated',
  component: MainLayout,
  beforeLoad: ({ location }) => {
    const { status } = useAuthStore.getState()
    if (status === 'unauthenticated') {
      throw redirect({
        to: '/login',
        search: { redirect: location.href },
      })
    }
  },
})

const filesRoute = createRoute({
  getParentRoute: () => authenticatedLayoutRoute,
  path: '/files',
  component: FileBrowserPage,
})

const folderRoute = createRoute({
  getParentRoute: () => authenticatedLayoutRoute,
  path: '/files/$folderId',
  component: FileBrowserPage,
})

const trashRoute = createRoute({
  getParentRoute: () => authenticatedLayoutRoute,
  path: '/trash',
  component: TrashPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => authenticatedLayoutRoute,
  path: '/settings',
  component: SettingsPage,
})

const groupsRoute = createRoute({
  getParentRoute: () => authenticatedLayoutRoute,
  path: '/groups',
  component: GroupsPage,
})

const groupDetailRoute = createRoute({
  getParentRoute: () => authenticatedLayoutRoute,
  path: '/groups/$groupId',
  component: GroupDetailPage,
})

// Index route redirects to /files
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/files' })
  },
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  authLayoutRoute.addChildren([loginRoute, registerRoute]),
  authenticatedLayoutRoute.addChildren([
    filesRoute,
    folderRoute,
    trashRoute,
    settingsRoute,
    groupsRoute,
    groupDetailRoute,
  ]),
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
