import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { toast } from 'sonner';
import { useGrantRoleMutation, useRevokeGrantMutation } from '../mutations';

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

describe('useGrantRoleMutation', () => {
  const resourceId = 'file-1';
  const input = {
    granteeType: 'user' as const,
    granteeId: 'user-uuid-1',
    role: 'contributor' as const,
  };

  describe('for file resource', () => {
    it('should call POST /files/{id}/permissions with correct params', async () => {
      mockApi.POST.mockResolvedValueOnce({
        data: { data: { id: 'perm-1' } },
        error: undefined,
        response: new Response(),
      } as never);

      const { result } = renderHook(
        () => useGrantRoleMutation('file', resourceId),
        { wrapper: createWrapper() },
      );

      await result.current.mutateAsync(input);

      expect(mockApi.POST).toHaveBeenCalledWith('/files/{id}/permissions', {
        params: { path: { id: resourceId } },
        body: input,
      });
    });

    it('should show success toast on success', async () => {
      mockApi.POST.mockResolvedValueOnce({
        data: { data: { id: 'perm-1' } },
        error: undefined,
        response: new Response(),
      } as never);

      const { result } = renderHook(
        () => useGrantRoleMutation('file', resourceId),
        { wrapper: createWrapper() },
      );

      await result.current.mutateAsync(input);

      await waitFor(() => {
        expect(mockToast.success).toHaveBeenCalledWith('Permission granted');
      });
    });

    it('should throw server error message on failure', async () => {
      mockApi.POST.mockResolvedValueOnce({
        data: undefined,
        error: { error: { message: 'Permission already exists' } },
        response: new Response(),
      } as never);

      const { result } = renderHook(
        () => useGrantRoleMutation('file', resourceId),
        { wrapper: createWrapper() },
      );

      await expect(result.current.mutateAsync(input)).rejects.toThrow(
        'Permission already exists',
      );
    });

    it('should throw default message when server error has no message', async () => {
      mockApi.POST.mockResolvedValueOnce({
        data: undefined,
        error: { error: {} },
        response: new Response(),
      } as never);

      const { result } = renderHook(
        () => useGrantRoleMutation('file', resourceId),
        { wrapper: createWrapper() },
      );

      await expect(result.current.mutateAsync(input)).rejects.toThrow(
        'Failed to grant permission',
      );
    });
  });

  describe('for folder resource', () => {
    it('should call POST /folders/{id}/permissions with correct params', async () => {
      mockApi.POST.mockResolvedValueOnce({
        data: { data: { id: 'perm-2' } },
        error: undefined,
        response: new Response(),
      } as never);

      const { result } = renderHook(
        () => useGrantRoleMutation('folder', resourceId),
        { wrapper: createWrapper() },
      );

      await result.current.mutateAsync(input);

      expect(mockApi.POST).toHaveBeenCalledWith('/folders/{id}/permissions', {
        params: { path: { id: resourceId } },
        body: input,
      });
    });
  });
});

describe('useRevokeGrantMutation', () => {
  const resourceType = 'file';
  const resourceId = 'file-1';
  const permissionId = 'perm-1';

  it('should call DELETE /permissions/{id} with correct params', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(
      () => useRevokeGrantMutation(resourceType, resourceId),
      { wrapper: createWrapper() },
    );

    await result.current.mutateAsync(permissionId);

    expect(mockApi.DELETE).toHaveBeenCalledWith('/permissions/{id}', {
      params: { path: { id: permissionId } },
    });
  });

  it('should show success toast on success', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(
      () => useRevokeGrantMutation(resourceType, resourceId),
      { wrapper: createWrapper() },
    );

    await result.current.mutateAsync(permissionId);

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith('Permission revoked');
    });
  });

  it('should throw server error message on failure', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: { message: 'Permission not found' } },
      response: new Response(),
    } as never);

    const { result } = renderHook(
      () => useRevokeGrantMutation(resourceType, resourceId),
      { wrapper: createWrapper() },
    );

    await expect(result.current.mutateAsync(permissionId)).rejects.toThrow(
      'Permission not found',
    );
  });

  it('should throw default message when server error has no message', async () => {
    mockApi.DELETE.mockResolvedValueOnce({
      data: undefined,
      error: { error: {} },
      response: new Response(),
    } as never);

    const { result } = renderHook(
      () => useRevokeGrantMutation(resourceType, resourceId),
      { wrapper: createWrapper() },
    );

    await expect(result.current.mutateAsync(permissionId)).rejects.toThrow(
      'Failed to revoke permission',
    );
  });
});
