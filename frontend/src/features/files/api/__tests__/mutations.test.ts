import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { toast } from 'sonner';
import {
  useCreateFolderMutation,
  useRenameMutation,
  useTrashMutation,
  useMoveFileMutation,
  useMoveFolderMutation,
  useAbortUploadMutation,
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

describe('useCreateFolderMutation', () => {
  const input = { name: 'New Folder', parentId: 'parent-1' };

  it('calls api.POST with correct path and body', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { id: 'folder-1', name: 'New Folder' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCreateFolderMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.POST).toHaveBeenCalledWith('/folders', {
      body: input,
    });
  });

  it('shows success toast on success', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { id: 'folder-1' } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCreateFolderMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Folder created');
    });
  });

  it('throws error with server message on failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Duplicate name' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCreateFolderMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Duplicate name',
    );
  });

  it('throws default message when server error has no message', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCreateFolderMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Failed to create folder',
    );
  });

  it('shows error toast on error', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Server error' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useCreateFolderMutation(), {
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

describe('useRenameMutation', () => {
  it('calls PATCH /folders/{id}/rename for folder type', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRenameMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({
      id: 'folder-1',
      name: 'Renamed',
      type: 'folder',
    });

    expect(mockApi.PATCH).toHaveBeenCalledWith('/folders/{id}/rename', {
      params: { path: { id: 'folder-1' } },
      body: { name: 'Renamed' },
    });
  });

  it('calls PATCH /files/{id}/rename for file type', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRenameMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({
      id: 'file-1',
      name: 'Renamed',
      type: 'file',
    });

    expect(mockApi.PATCH).toHaveBeenCalledWith('/files/{id}/rename', {
      params: { path: { id: 'file-1' } },
      body: { name: 'Renamed' },
    });
  });

  it('shows success toast with correct entity type', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRenameMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({
      id: 'folder-1',
      name: 'Renamed',
      type: 'folder',
    });

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Folder renamed');
    });
  });

  it('throws error on folder rename failure', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRenameMutation(), {
      wrapper: createWrapper(),
    });

    await expect(
      result.current.mutateAsync({
        id: 'folder-1',
        name: 'X',
        type: 'folder',
      }),
    ).rejects.toThrow('Not found');
  });

  it('throws default message on file rename failure without message', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useRenameMutation(), {
      wrapper: createWrapper(),
    });

    await expect(
      result.current.mutateAsync({ id: 'file-1', name: 'X', type: 'file' }),
    ).rejects.toThrow('Failed to rename file');
  });
});

describe('useTrashMutation', () => {
  it('calls POST /files/{id}/trash for file type', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({ id: 'file-1', type: 'file' });

    expect(mockApi.POST).toHaveBeenCalledWith('/files/{id}/trash', {
      params: { path: { id: 'file-1' } },
    });
  });

  it('calls DELETE /folders/{id} for folder type', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({ id: 'folder-1', type: 'folder' });

    expect(mockApi.DELETE).toHaveBeenCalledWith('/folders/{id}', {
      params: { path: { id: 'folder-1' } },
    });
  });

  it('shows success toast for file trash', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({ id: 'file-1', type: 'file' });

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('File moved to trash');
    });
  });

  it('shows success toast for folder trash', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync({ id: 'folder-1', type: 'folder' });

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Folder moved to trash');
    });
  });

  it('throws error on file trash failure', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Permission denied' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashMutation(), {
      wrapper: createWrapper(),
    });

    await expect(
      result.current.mutateAsync({ id: 'file-1', type: 'file' }),
    ).rejects.toThrow('Permission denied');
  });

  it('throws default message on folder trash failure without message', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashMutation(), {
      wrapper: createWrapper(),
    });

    await expect(
      result.current.mutateAsync({ id: 'folder-1', type: 'folder' }),
    ).rejects.toThrow('Failed to delete folder');
  });
});

describe('useMoveFileMutation', () => {
  const input = { id: 'file-1', newFolderId: 'folder-2' };

  it('calls PATCH /files/{id}/move with correct params', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFileMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.PATCH).toHaveBeenCalledWith('/files/{id}/move', {
      params: { path: { id: 'file-1' } },
      body: { newFolderId: 'folder-2' },
    });
  });

  it('shows success toast on success', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFileMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('File moved');
    });
  });

  it('throws error on failure', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFileMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Not found',
    );
  });

  it('throws default message on failure without message', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFileMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Failed to move file',
    );
  });
});

describe('useMoveFolderMutation', () => {
  const input = { id: 'folder-1', newParentId: 'folder-2' };

  it('calls PATCH /folders/{id}/move with correct params', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFolderMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    expect(mockApi.PATCH).toHaveBeenCalledWith('/folders/{id}/move', {
      params: { path: { id: 'folder-1' } },
      body: { newParentId: 'folder-2' },
    });
  });

  it('shows success toast on success', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFolderMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(input);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Folder moved');
    });
  });

  it('throws error on failure', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Circular reference' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFolderMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Circular reference',
    );
  });

  it('throws default message on failure without message', async () => {
    mockApi.PATCH.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useMoveFolderMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(input)).rejects.toThrow(
      'Failed to move folder',
    );
  });
});

describe('useAbortUploadMutation', () => {
  const sessionId = 'session-123';

  it('calls DELETE /files/upload/{sessionId}', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAbortUploadMutation(), {
      wrapper: createWrapper(),
    });

    await result.current.mutateAsync(sessionId);

    expect(mockApi.DELETE).toHaveBeenCalledWith('/files/upload/{sessionId}', {
      params: { path: { sessionId } },
    });
  });

  it('throws error on failure', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Session not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAbortUploadMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(sessionId)).rejects.toThrow(
      'Session not found',
    );
  });

  it('throws default message on failure without message', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useAbortUploadMutation(), {
      wrapper: createWrapper(),
    });

    await expect(result.current.mutateAsync(sessionId)).rejects.toThrow(
      'Failed to abort upload',
    );
  });
});
