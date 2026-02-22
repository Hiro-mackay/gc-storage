import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { useGroupInvitations, usePendingInvitations } from '../queries';

vi.mock('@/lib/api/client', () => ({
  api: {
    GET: vi.fn(),
    POST: vi.fn(),
    PATCH: vi.fn(),
    DELETE: vi.fn(),
  },
}));

const mockApi = vi.mocked(api);

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useGroupInvitations', () => {
  const groupId = 'group-1';
  const mockInvitations = [
    { id: 'inv-1', email: 'a@example.com', role: 'viewer', expiresAt: null },
    {
      id: 'inv-2',
      email: 'b@example.com',
      role: 'contributor',
      expiresAt: '2026-01-01',
    },
  ];

  it('should call GET /groups/{id}/invitations with correct groupId', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockInvitations },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useGroupInvitations(groupId), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/groups/{id}/invitations', {
      params: { path: { id: groupId } },
    });
  });

  it('should return invitation list on success', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockInvitations },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useGroupInvitations(groupId), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockInvitations);
  });

  it('should return empty array when data is null', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: null },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useGroupInvitations(groupId), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual([]);
  });

  it('should throw on API error', async () => {
    const apiError = { error: { message: 'Forbidden' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useGroupInvitations(groupId), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('usePendingInvitations', () => {
  const mockPending = [
    {
      invitation: { id: 'inv-1', role: 'viewer', expiresAt: null },
      group: { id: 'g1', name: 'Group A' },
    },
  ];

  it('should call GET /invitations/pending', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockPending },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePendingInvitations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/invitations/pending');
  });

  it('should return pending invitation list on success', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockPending },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePendingInvitations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockPending);
  });

  it('should return empty array when data is null', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: null },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePendingInvitations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual([]);
  });

  it('should throw on API error', async () => {
    const apiError = { error: { message: 'Unauthorized' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePendingInvitations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});
