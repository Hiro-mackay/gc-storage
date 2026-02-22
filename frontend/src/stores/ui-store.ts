import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type ViewMode = 'list' | 'grid';
type SortBy = 'name' | 'updatedAt' | 'size' | 'type';
type SortOrder = 'asc' | 'desc';
type Theme = 'light' | 'dark' | 'system';

interface UIState {
  sidebarOpen: boolean;
  viewMode: ViewMode;
  sortBy: SortBy;
  sortOrder: SortOrder;
  theme: Theme;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;
  setViewMode: (mode: ViewMode) => void;
  setSortBy: (by: SortBy) => void;
  setSortOrder: (order: SortOrder) => void;
  setTheme: (theme: Theme) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      viewMode: 'list',
      sortBy: 'name',
      sortOrder: 'asc',
      theme: 'system',
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      toggleSidebar: () =>
        set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      setViewMode: (mode) => set({ viewMode: mode }),
      setSortBy: (by) => set({ sortBy: by }),
      setSortOrder: (order) => set({ sortOrder: order }),
      setTheme: (theme) => set({ theme }),
    }),
    { name: 'gc-storage-ui-settings' },
  ),
);
