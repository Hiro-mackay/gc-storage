import { Outlet } from '@tanstack/react-router';

export function AuthLayout() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-md space-y-6 px-4">
        <div className="text-center">
          <h1 className="text-2xl font-bold">GC Storage</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Cloud storage for your team
          </p>
        </div>
        <Outlet />
      </div>
    </div>
  );
}
