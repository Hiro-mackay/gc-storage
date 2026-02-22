import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { useAuthStore } from '@/stores/auth-store';
import { createWrapper } from '@/test/test-utils';
import {
  useLoginMutation,
  useRegisterMutation,
  useOAuthLoginMutation,
  useVerifyEmailMutation,
  useForgotPasswordMutation,
  useResetPasswordMutation,
  useChangePasswordMutation,
  useLogoutMutation,
  useResendVerificationMutation,
} from '../mutations';

vi.mock('@/lib/api/client', () => ({
  api: {
    GET: vi.fn(),
    POST: vi.fn(),
    PATCH: vi.fn(),
    DELETE: vi.fn(),
  },
}));

const mockApi = vi.mocked(api);

const mockUser = {
  id: 'user-1',
  email: 'test@example.com',
  name: 'Test User',
} as Parameters<ReturnType<(typeof useAuthStore)['getState']>['setUser']>[0];

beforeEach(() => {
  vi.clearAllMocks();
  useAuthStore.setState({ status: 'initializing', user: null });
});

describe('useLoginMutation', () => {
  const input = { email: 'test@example.com', password: 'password123' };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { user: mockUser } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLoginMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/login', {
      body: input,
    });
  });

  it('sets auth store to authenticated on success with user', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { user: mockUser } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLoginMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      const state = useAuthStore.getState();
      expect(state.status).toBe('authenticated');
      expect(state.user).toEqual(mockUser);
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Invalid credentials' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLoginMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Invalid credentials',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLoginMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Login failed',
    );
  });
});

describe('useRegisterMutation', () => {
  const input = {
    email: 'new@example.com',
    password: 'password123',
    name: 'New User',
  };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { user: mockUser } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRegisterMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/register', {
      body: input,
    });
  });

  it('sets auth store to authenticated on success with user', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { user: mockUser } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRegisterMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      const state = useAuthStore.getState();
      expect(state.status).toBe('authenticated');
      expect(state.user).toEqual(mockUser);
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Email already exists' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRegisterMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Email already exists',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRegisterMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Registration failed',
    );
  });
});

describe('useOAuthLoginMutation', () => {
  const input = { provider: 'google', code: 'auth-code-123' };

  it('calls api.POST with correct path, params, and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { user: mockUser } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useOAuthLoginMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/oauth/{provider}', {
      params: { path: { provider: 'google' } },
      body: { code: 'auth-code-123' },
    });
  });

  it('sets auth store to authenticated on success with user', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { user: mockUser } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useOAuthLoginMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      const state = useAuthStore.getState();
      expect(state.status).toBe('authenticated');
      expect(state.user).toEqual(mockUser);
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'OAuth failed' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useOAuthLoginMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'OAuth failed',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useOAuthLoginMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Authentication failed',
    );
  });
});

describe('useVerifyEmailMutation', () => {
  const token = 'verify-token-123';

  it('calls api.POST with correct path and query params', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useVerifyEmailMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(token);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/email/verify', {
      params: { query: { token } },
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Token expired' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useVerifyEmailMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(token)).rejects.toThrow(
      'Token expired',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useVerifyEmailMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(token)).rejects.toThrow(
      'Verification failed',
    );
  });
});

describe('useForgotPasswordMutation', () => {
  const input = { email: 'test@example.com' };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { message: 'Email sent' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useForgotPasswordMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/password/forgot', {
      body: input,
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'User not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useForgotPasswordMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'User not found',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useForgotPasswordMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Request failed',
    );
  });
});

describe('useResetPasswordMutation', () => {
  const input = { token: 'reset-token', password: 'newpass123' };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { message: 'Password reset' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useResetPasswordMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/password/reset', {
      body: input,
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Invalid token' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useResetPasswordMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Invalid token',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useResetPasswordMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Reset failed',
    );
  });
});

describe('useChangePasswordMutation', () => {
  const input = { current_password: 'oldpass', new_password: 'newpass123' };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { message: 'Password changed' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useChangePasswordMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/password/change', {
      body: input,
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Wrong password' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useChangePasswordMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Wrong password',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useChangePasswordMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Password change failed',
    );
  });
});

describe('useLogoutMutation', () => {
  it('calls api.POST with correct path', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLogoutMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync();

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/logout');
  });

  it('clears auth store on success', async () => {
    useAuthStore.setState({ status: 'authenticated', user: mockUser });

    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLogoutMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync();

    await waitFor(() => {
      const state = useAuthStore.getState();
      expect(state.status).toBe('unauthenticated');
      expect(state.user).toBeNull();
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Session expired' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLogoutMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync()).rejects.toThrow(
      'Session expired',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useLogoutMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync()).rejects.toThrow('Logout failed');
  });
});

describe('useResendVerificationMutation', () => {
  const input = { email: 'test@example.com' };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { message: 'Verification email sent' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useResendVerificationMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/auth/email/resend', {
      body: input,
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Too many requests' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useResendVerificationMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Too many requests',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useResendVerificationMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Resend verification failed',
    );
  });
});
