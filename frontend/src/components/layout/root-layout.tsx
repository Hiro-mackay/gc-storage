import { useEffect, useRef } from 'react';
import { Outlet } from '@tanstack/react-router';
import { QueryClientProvider } from '@tanstack/react-query';
import { Loader2 } from 'lucide-react';
import { Toaster } from '@/components/ui/sonner';
import { queryClient } from '@/app/router';
import { fetchCurrentUser } from '@/features/auth/api/queries';
import { useAuthStore } from '@/stores/auth-store';

function AuthInitializer({ children }: { children: React.ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const setUser = useAuthStore((s) => s.setUser);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const checkedRef = useRef(false);

  useEffect(() => {
    if (checkedRef.current) return;
    checkedRef.current = true;

    const checkAuth = async () => {
      const user = await fetchCurrentUser();
      if (user) {
        setUser(user);
      } else {
        clearAuth();
      }
    };

    checkAuth();
  }, [setUser, clearAuth]);

  if (status === 'initializing') {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return <>{children}</>;
}

export function RootLayout() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthInitializer>
        <Outlet />
      </AuthInitializer>
      <Toaster />
    </QueryClientProvider>
  );
}
