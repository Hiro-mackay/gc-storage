import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { toast } from 'sonner';
import {
  useInviteMemberMutation,
  useAcceptInvitationMutation,
  useDeclineInvitationMutation,
  useCancelInvitationMutation,
} from '../mutations';

vi.mock('@/lib/api/client', () => ({
  api: {
    GET: vi.fn(),
    POST: vi.fn(),
    PATCH: vi.fn(),
    DELETE: vi.fn(),
  },
}));

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

const mockApi = vi.mocked(api);
const mockToast = vi.mocked(toast);

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useInviteMemberMutation', () => {
  const groupId = 'group-1';
  const input = { email: 'user@example.com', role: 'viewer' as const };

  it('should call POST /groups/{id}/invitations with correct params', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { id: 'inv-1' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useInviteMemberMutation(groupId), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/groups/{id}/invitations', {
      params: { path: { id: groupId } },
      body: { email: input.email, role: input.role },
    });
  });

  it('should show success toast on success', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { id: 'inv-1' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useInviteMemberMutation(groupId), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Invitation sent');
    });
  });

  it('should throw server error message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Email already invited' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useInviteMemberMutation(groupId), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Email already invited',
    );
  });

  it('should throw default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useInviteMemberMutation(groupId), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Failed to send invitation',
    );
  });

  it('should show error toast on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Server error' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useInviteMemberMutation(groupId), {
      wrapper: createWrapper(),
    });

    try {
      await result.current.mutateAsync(input);
    } catch {
      // expected
    }

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith('Server error');
    });
  });
});

describe('useAcceptInvitationMutation', () => {
  const token = 'abc123token';

  it('should call POST /invitations/{token}/accept', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { group: { id: 'g1', name: 'Group A' } } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAcceptInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(token);

    expect(mockApi.POST).toHaveBeenCalledWith('/invitations/{token}/accept', {
      params: { path: { token } },
    });
  });

  it('should show success toast on success', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { group: { id: 'g1' } } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAcceptInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(token);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Invitation accepted');
    });
  });

  it('should throw server error message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Token expired' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAcceptInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(token)).rejects.toThrow(
      'Token expired',
    );
  });

  it('should throw default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAcceptInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(token)).rejects.toThrow(
      'Failed to accept invitation',
    );
  });
});

describe('useDeclineInvitationMutation', () => {
  const token = 'abc123token';

  it('should call POST /invitations/{token}/decline', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useDeclineInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(token);

    expect(mockApi.POST).toHaveBeenCalledWith('/invitations/{token}/decline', {
      params: { path: { token } },
    });
  });

  it('should show success toast on success', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useDeclineInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(token);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Invitation declined');
    });
  });

  it('should throw server error message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Token not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useDeclineInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(token)).rejects.toThrow(
      'Token not found',
    );
  });

  it('should throw default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useDeclineInvitationMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(token)).rejects.toThrow(
      'Failed to decline invitation',
    );
  });
});

describe('useCancelInvitationMutation', () => {
  const groupId = 'group-1';
  const invitationId = 'inv-1';

  it('should call DELETE /groups/{id}/invitations/{invitationId}', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCancelInvitationMutation(groupId), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(invitationId);

    expect(mockApi.DELETE).toHaveBeenCalledWith(
      '/groups/{id}/invitations/{invitationId}',
      {
        params: { path: { id: groupId, invitationId } },
      },
    );
  });

  it('should show success toast on success', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCancelInvitationMutation(groupId), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(invitationId);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Invitation cancelled');
    });
  });

  it('should throw server error message on failure', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCancelInvitationMutation(groupId), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(invitationId)).rejects.toThrow(
      'Not found',
    );
  });

  it('should throw default message when server error has no message', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCancelInvitationMutation(groupId), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(invitationId)).rejects.toThrow(
      'Failed to cancel invitation',
    );
  });
});
