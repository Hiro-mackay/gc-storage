import { renderHook } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { useUploadStore } from '@/stores/upload-store';
import { toast } from 'sonner';
import { useFileUpload } from '../use-file-upload';

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

vi.stubGlobal('crypto', { randomUUID: () => 'test-uuid' });

interface MockXHRInstance {
  open: ReturnType<typeof vi.fn>;
  send: ReturnType<typeof vi.fn>;
  setRequestHeader: ReturnType<typeof vi.fn>;
  getResponseHeader: ReturnType<typeof vi.fn>;
  upload: { onprogress: ((e: ProgressEvent) => void) | null };
  onload: (() => void) | null;
  onerror: (() => void) | null;
  status: number;
}

let mockXhrInstance: MockXHRInstance;

function createMockXhr(): MockXHRInstance {
  return {
    open: vi.fn(),
    send: vi.fn(),
    setRequestHeader: vi.fn(),
    getResponseHeader: vi.fn(),
    upload: { onprogress: null },
    onload: null,
    onerror: null,
    status: 200,
  };
}

function MockXHR() {
  const instance = createMockXhr();
  mockXhrInstance = instance;
  return instance;
}

beforeEach(() => {
  vi.clearAllMocks();
  useUploadStore.setState({ uploads: new Map() });
  vi.stubGlobal('XMLHttpRequest', MockXHR);
});

function createFile(name = 'test.pdf', size = 1024, type = 'application/pdf') {
  return new File(['x'.repeat(size)], name, { type });
}

describe('useFileUpload', () => {
  const folderId = 'folder-1';
  const uploadUrl = 'https://storage.example.com/upload?signed=abc';

  function mockInitiateSuccess(fileId = 'uploaded-file-1') {
    mockApi.POST.mockResolvedValueOnce({
      data: {
        data: {
          fileId,
          uploadUrls: [{ url: uploadUrl }],
        },
      },
      error: undefined,
      response: new Response(),
    } as never);
  }

  function mockCompleteSuccess() {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: {} },
      error: undefined,
      response: new Response(),
    } as never);
  }

  it('completes full upload flow successfully', async () => {
    mockInitiateSuccess();
    mockCompleteSuccess();

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    mockXhrInstance.status = 200;
    mockXhrInstance.getResponseHeader.mockReturnValue('"etag-123"');
    mockXhrInstance.onload!();

    await promise;

    expect(mockApi.POST).toHaveBeenCalledWith('/files/upload', {
      body: {
        fileName: 'test.pdf',
        folderId,
        mimeType: 'application/pdf',
        size: 1024,
      },
    });

    expect(mockApi.POST).toHaveBeenCalledWith('/files/upload/complete', {
      body: {
        storageKey: 'uploaded-file-1',
        etag: '"etag-123"',
        size: 1024,
        minioVersionId: '',
      },
    });

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.status).toBe('completed');
    expect(upload?.progress).toBe(100);
  });

  it('adds upload to store on start', async () => {
    mockInitiateSuccess();
    mockCompleteSuccess();

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile('doc.txt', 2048);
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.fileName).toBe('doc.txt');
    expect(upload?.fileSize).toBe(2048);

    mockXhrInstance.status = 200;
    mockXhrInstance.getResponseHeader.mockReturnValue('"etag"');
    mockXhrInstance.onload!();
    await promise;
  });

  it('sets status to failed when initiate fails', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Quota exceeded' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    await result.current.uploadFile(file, folderId);

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.status).toBe('failed');
    expect(upload?.error).toBe('Quota exceeded');
    expect(mockToast.error).toHaveBeenCalledWith('test.pdf: Quota exceeded');
  });

  it('sets status to failed when no upload URL is returned', async () => {
    mockApi.POST.mockResolvedValueOnce({
      data: { data: { fileId: 'f1', uploadUrls: [] } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    await result.current.uploadFile(file, folderId);

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.status).toBe('failed');
    expect(upload?.error).toBe('No upload URL received');
  });

  it('sets status to failed when XHR fails with error status', async () => {
    mockInitiateSuccess();

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    mockXhrInstance.status = 500;
    mockXhrInstance.onload!();

    await promise;

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.status).toBe('failed');
    expect(upload?.error).toBe('Storage upload failed: 500');
  });

  it('sets status to failed on XHR network error', async () => {
    mockInitiateSuccess();

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    mockXhrInstance.onerror!();

    await promise;

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.status).toBe('failed');
    expect(upload?.error).toBe('Network error during upload');
  });

  it('sets status to failed when complete call fails', async () => {
    mockInitiateSuccess();
    mockApi.POST.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Complete failed' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    mockXhrInstance.status = 200;
    mockXhrInstance.getResponseHeader.mockReturnValue('"etag"');
    mockXhrInstance.onload!();

    await promise;

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    expect(upload?.status).toBe('failed');
    expect(upload?.error).toBe('Failed to finalize upload');
  });

  it('updates progress via XHR onprogress', async () => {
    mockInitiateSuccess();
    mockCompleteSuccess();

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile();
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    mockXhrInstance.upload.onprogress!({
      lengthComputable: true,
      loaded: 512,
      total: 1024,
    } as ProgressEvent);

    const upload = useUploadStore.getState().uploads.get('test-uuid');
    // 512/1024 * 95 = 47.5 -> rounded to 48
    expect(upload?.progress).toBe(48);

    mockXhrInstance.status = 200;
    mockXhrInstance.getResponseHeader.mockReturnValue('"etag"');
    mockXhrInstance.onload!();
    await promise;
  });

  it('uses application/octet-stream when file has no type', async () => {
    mockInitiateSuccess();
    mockCompleteSuccess();

    const { result } = renderHook(() => useFileUpload(), {
      wrapper: createWrapper(),
    });

    const file = createFile('data.bin', 100, '');
    const promise = result.current.uploadFile(file, folderId);

    await vi.waitFor(() => {
      expect(mockXhrInstance.send).toHaveBeenCalled();
    });

    expect(mockApi.POST).toHaveBeenCalledWith('/files/upload', {
      body: expect.objectContaining({
        mimeType: 'application/octet-stream',
      }),
    });

    expect(mockXhrInstance.setRequestHeader).toHaveBeenCalledWith(
      'Content-Type',
      'application/octet-stream',
    );

    mockXhrInstance.status = 200;
    mockXhrInstance.getResponseHeader.mockReturnValue('"etag"');
    mockXhrInstance.onload!();
    await promise;
  });
});
