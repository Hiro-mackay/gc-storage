import { beforeEach, describe, expect, it, vi } from 'vitest';

const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] ?? null),
  };
})();

Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock });

// Import store after localStorage mock is set up
const { useUIStore } = await import('@/stores/ui-store');

describe('useUIStore', () => {
  beforeEach(() => {
    useUIStore.setState({
      sidebarOpen: true,
      viewMode: 'list',
      sortBy: 'name',
      sortOrder: 'asc',
      theme: 'system',
    });
  });

  it('has correct initial state', () => {
    const state = useUIStore.getState();
    expect(state.sidebarOpen).toBe(true);
    expect(state.viewMode).toBe('list');
    expect(state.sortBy).toBe('name');
    expect(state.sortOrder).toBe('asc');
    expect(state.theme).toBe('system');
  });

  it('setSidebarOpen sets sidebar state', () => {
    useUIStore.getState().setSidebarOpen(false);
    expect(useUIStore.getState().sidebarOpen).toBe(false);

    useUIStore.getState().setSidebarOpen(true);
    expect(useUIStore.getState().sidebarOpen).toBe(true);
  });

  it('toggleSidebar flips sidebar state', () => {
    expect(useUIStore.getState().sidebarOpen).toBe(true);

    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().sidebarOpen).toBe(false);

    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().sidebarOpen).toBe(true);
  });

  it('setViewMode updates view mode', () => {
    useUIStore.getState().setViewMode('grid');
    expect(useUIStore.getState().viewMode).toBe('grid');

    useUIStore.getState().setViewMode('list');
    expect(useUIStore.getState().viewMode).toBe('list');
  });

  it('setSortBy updates sort field', () => {
    useUIStore.getState().setSortBy('updatedAt');
    expect(useUIStore.getState().sortBy).toBe('updatedAt');

    useUIStore.getState().setSortBy('size');
    expect(useUIStore.getState().sortBy).toBe('size');
  });

  it('setSortOrder updates sort order', () => {
    useUIStore.getState().setSortOrder('desc');
    expect(useUIStore.getState().sortOrder).toBe('desc');

    useUIStore.getState().setSortOrder('asc');
    expect(useUIStore.getState().sortOrder).toBe('asc');
  });

  it('setTheme updates theme', () => {
    useUIStore.getState().setTheme('dark');
    expect(useUIStore.getState().theme).toBe('dark');

    useUIStore.getState().setTheme('light');
    expect(useUIStore.getState().theme).toBe('light');
  });
});
