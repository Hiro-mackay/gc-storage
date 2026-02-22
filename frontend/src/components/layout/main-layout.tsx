import { useEffect } from 'react';
import {
  Outlet,
  Link,
  useMatchRoute,
  useNavigate,
} from '@tanstack/react-router';
import { useAuthStore } from '@/stores/auth-store';
import { useUIStore } from '@/stores/ui-store';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Files, Trash2, Settings, Menu, LogOut, Users } from 'lucide-react';
import { api } from '@/lib/api/client';
import { cn } from '@/lib/utils';

const navItems = [
  { to: '/files' as const, label: 'My Files', icon: Files },
  { to: '/groups' as const, label: 'Groups', icon: Users },
  { to: '/trash' as const, label: 'Trash', icon: Trash2 },
  { to: '/settings' as const, label: 'Settings', icon: Settings },
];

export function MainLayout() {
  const { user, clearAuth } = useAuthStore();
  const { sidebarOpen, toggleSidebar, theme } = useUIStore();
  const matchRoute = useMatchRoute();
  const navigate = useNavigate();

  useEffect(() => {
    const root = document.documentElement;
    if (theme === 'dark') {
      root.classList.add('dark');
    } else if (theme === 'light') {
      root.classList.remove('dark');
    } else {
      // system
      const mq = window.matchMedia('(prefers-color-scheme: dark)');
      const apply = () => {
        root.classList.toggle('dark', mq.matches);
      };
      apply();
      mq.addEventListener('change', apply);
      return () => mq.removeEventListener('change', apply);
    }
  }, [theme]);

  const handleLogout = async () => {
    await api.POST('/auth/logout');
    clearAuth();
    navigate({ to: '/login' });
  };

  const initials = user?.name
    ? user.name
        .split(' ')
        .map((n) => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2)
    : '??';

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex flex-col border-r bg-muted/30 transition-all duration-200',
          sidebarOpen ? 'w-60' : 'w-0 overflow-hidden',
        )}
      >
        <div className="flex h-14 items-center px-4">
          <Link to="/files" className="font-bold text-lg">
            GC Storage
          </Link>
        </div>
        <Separator />
        <nav className="flex-1 space-y-1 px-2 py-2">
          {navItems.map((item) => {
            const isActive = matchRoute({ to: item.to, fuzzy: true });
            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-accent text-accent-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                )}
              >
                <item.icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <header className="flex h-14 items-center justify-between border-b px-4">
          <Button variant="ghost" size="icon" onClick={toggleSidebar}>
            <Menu className="h-5 w-5" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="gap-2">
                <Avatar className="h-7 w-7">
                  <AvatarFallback className="text-xs">
                    {initials}
                  </AvatarFallback>
                </Avatar>
                <span className="text-sm">{user?.name}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleLogout}>
                <LogOut className="mr-2 h-4 w-4" />
                Logout
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
