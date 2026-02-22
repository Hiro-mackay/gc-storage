import { useAuthStore } from '@/stores/auth-store';

const mockUser = {
  id: 'user-1',
  email: 'test@example.com',
  name: 'Test User',
} as Parameters<ReturnType<(typeof useAuthStore)['getState']>['setUser']>[0];

describe('useAuthStore', () => {
  beforeEach(() => {
    useAuthStore.setState({ status: 'initializing', user: null });
  });

  it('has correct initial state', () => {
    const state = useAuthStore.getState();
    expect(state.status).toBe('initializing');
    expect(state.user).toBeNull();
  });

  it('setUser sets status to authenticated and stores user', () => {
    useAuthStore.getState().setUser(mockUser);
    const state = useAuthStore.getState();
    expect(state.status).toBe('authenticated');
    expect(state.user).toEqual(mockUser);
  });

  it('clearAuth sets status to unauthenticated and clears user', () => {
    useAuthStore.getState().setUser(mockUser);
    useAuthStore.getState().clearAuth();
    const state = useAuthStore.getState();
    expect(state.status).toBe('unauthenticated');
    expect(state.user).toBeNull();
  });

  it('setInitializing sets status back to initializing', () => {
    useAuthStore.getState().setUser(mockUser);
    useAuthStore.getState().setInitializing();
    expect(useAuthStore.getState().status).toBe('initializing');
  });

  it('handles full flow: initializing -> authenticated -> unauthenticated', () => {
    expect(useAuthStore.getState().status).toBe('initializing');

    useAuthStore.getState().setUser(mockUser);
    expect(useAuthStore.getState().status).toBe('authenticated');
    expect(useAuthStore.getState().user).toEqual(mockUser);

    useAuthStore.getState().clearAuth();
    expect(useAuthStore.getState().status).toBe('unauthenticated');
    expect(useAuthStore.getState().user).toBeNull();
  });
});
