import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import {
  useFolderContents,
  useFolderAncestors,
  useFileVersions,
  useUploadStatus,
} from '../queries';

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

describe('useFolderContents', () => {
  const mockContents = {
    folders: [{ id: 'f1', name: 'Folder 1' }],
    files: [{ id: 'file1', name: 'File 1' }],
  };

  it('calls GET /folders/{id}/contents with folderId', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockContents },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFolderContents('folder-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/folders/{id}/contents', {
      params: { path: { id: 'folder-1' } },
    });
    expect(result.current.data).toEqual(mockContents);
  });

  it('uses "root" when folderId is null', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockContents },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFolderContents(null), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/folders/{id}/contents', {
      params: { path: { id: 'root' } },
    });
  });

  it('throws on API error', async () => {
    const apiError = { error: { message: 'Not found' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFolderContents('bad-id'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('useFolderAncestors', () => {
  const mockAncestors = [
    { id: 'root', name: 'Root' },
    { id: 'parent', name: 'Parent' },
  ];

  it('calls GET /folders/{id}/ancestors and returns items', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: { items: mockAncestors } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFolderAncestors('folder-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/folders/{id}/ancestors', {
      params: { path: { id: 'folder-1' } },
    });
    expect(result.current.data).toEqual(mockAncestors);
  });

  it('is disabled and does not fetch when folderId is null', async () => {
    const { result } = renderHook(() => useFolderAncestors(null), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockApi.GET).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();
  });

  it('throws on API error', async () => {
    const apiError = { error: { message: 'Server error' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFolderAncestors('bad-id'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('useFileVersions', () => {
  const mockVersions = [
    { id: 'v1', version: 1 },
    { id: 'v2', version: 2 },
  ];

  it('calls GET /files/{id}/versions and returns versions', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: { versions: mockVersions } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFileVersions('file-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/files/{id}/versions', {
      params: { path: { id: 'file-1' } },
    });
    expect(result.current.data).toEqual(mockVersions);
  });

  it('throws on API error', async () => {
    const apiError = { error: { message: 'Not found' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFileVersions('bad-id'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('useUploadStatus', () => {
  const mockStatus = { sessionId: 's1', status: 'uploading', progress: 50 };

  it('calls GET /files/upload/{sessionId} and returns data', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockStatus },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useUploadStatus('session-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/files/upload/{sessionId}', {
      params: { path: { sessionId: 'session-1' } },
    });
    expect(result.current.data).toEqual(mockStatus);
  });

  it('is disabled when sessionId is empty', async () => {
    const { result } = renderHook(() => useUploadStatus(''), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockApi.GET).not.toHaveBeenCalled();
  });

  it('throws on API error', async () => {
    const apiError = { error: { message: 'Not found' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useUploadStatus('bad-session'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});
