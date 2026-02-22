import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { toast } from 'sonner';
import {
  useRestoreFileMutation,
  usePermanentDeleteMutation,
  useEmptyTrashMutation,
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

describe('useRestoreFileMutation', () => {
  it('calls POST /trash/files/{id}/restore with correct params', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { message: 'restored' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRestoreFileMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync('file-1');

    expect(mockApi.POST).toHaveBeenCalledWith(
      '/trash/files/{id}/restore',
      expect.objectContaining({
        params: { path: { id: 'file-1' } },
        body: {},
      }),
    );
  });

  it('shows success toast on success', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: {} },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRestoreFileMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync('file-1');

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('File restored');
    });
  });

  it('shows error toast on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRestoreFileMutation(), {
      wrapper: createWrapper(),
    });

    try {
      await result.current.mutateAsync('bad-id');
    } catch {
      // expected
    }

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith('Failed to restore file');
    });
  });
});

describe('usePermanentDeleteMutation', () => {
  it('calls DELETE /trash/files/{id} with correct params', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePermanentDeleteMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync('file-1');

    expect(mockApi.DELETE).toHaveBeenCalledWith(
      '/trash/files/{id}',
      expect.objectContaining({
        params: { path: { id: 'file-1' } },
      }),
    );
  });

  it('shows success toast on success', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePermanentDeleteMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync('file-1');

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Permanently deleted');
    });
  });

  it('shows error toast on failure', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Forbidden' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => usePermanentDeleteMutation(), {
      wrapper: createWrapper(),
    });

    try {
      await result.current.mutateAsync('bad-id');
    } catch {
      // expected
    }

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith('Failed to delete');
    });
  });
});

describe('useEmptyTrashMutation', () => {
  it('calls DELETE /trash', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: { data: { message: 'accepted', deletedCount: 5 } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useEmptyTrashMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync();

    expect(mockApi.DELETE).toHaveBeenCalledWith('/trash');
  });

  it('shows success toast on success', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: { data: { message: 'accepted' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useEmptyTrashMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync();

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Trash emptied');
    });
  });

  it('shows error toast on failure', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Server error' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useEmptyTrashMutation(), {
      wrapper: createWrapper(),
    });

    try {
      await result.current.mutateAsync();
    } catch {
      // expected
    }

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith('Failed to empty trash');
    });
  });
});
