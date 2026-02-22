import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { shareKeys } from '@/lib/api/queries';
import { toast } from 'sonner';

export function useAccessShareLinkMutation(token: string) {
  return useMutation({
    mutationFn: async (password?: string) => {
      const { data, error } = await api.POST('/share/{token}/access', {
        params: { path: { token } },
        body: { password: password ?? '' },
      });
      if (error) throw error;
      return data?.data ?? null;
    },
  });
}

export function useCreateShareLinkMutation(
  resourceType: 'file' | 'folder',
  resourceId: string,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      permission: 'read' | 'write';
      password?: string;
      expiresAt?: string;
      maxAccessCount?: number;
    }) => {
      if (resourceType === 'file') {
        const { data, error } = await api.POST('/files/{id}/share', {
          params: { path: { id: resourceId } },
          body: {
            permission: input.permission,
            password: input.password,
            expiresAt: input.expiresAt,
            maxAccessCount: input.maxAccessCount,
          },
        });
        if (error) throw new Error('Failed to create share link');
        return data?.data ?? null;
      } else {
        const { data, error } = await api.POST('/folders/{id}/share', {
          params: { path: { id: resourceId } },
          body: {
            permission: input.permission,
            password: input.password,
            expiresAt: input.expiresAt,
            maxAccessCount: input.maxAccessCount,
          },
        });
        if (error) throw new Error('Failed to create share link');
        return data?.data ?? null;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: shareKeys.list(resourceType, resourceId),
      });
      toast.success('Share link created');
    },
    onError: () => {
      toast.error('Failed to create share link');
    },
  });
}

export function useUpdateShareLinkMutation(shareLinkId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      permission?: 'read' | 'write';
      password?: string;
      expiresAt?: string;
      maxAccessCount?: number;
    }) => {
      const { data, error } = await api.PATCH('/share-links/{id}', {
        params: { path: { id: shareLinkId } },
        body: input,
      });
      if (error) throw new Error('Failed to update share link');
      return data?.data ?? null;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: shareKeys.all });
      toast.success('Share link updated');
    },
    onError: () => {
      toast.error('Failed to update share link');
    },
  });
}

export function useRevokeShareLinkMutation(
  resourceType?: string,
  resourceId?: string,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (shareLinkId: string) => {
      const { error } = await api.DELETE('/share-links/{id}', {
        params: { path: { id: shareLinkId } },
      });
      if (error) throw new Error('Failed to revoke share link');
    },
    onSuccess: () => {
      if (resourceType && resourceId) {
        queryClient.invalidateQueries({
          queryKey: shareKeys.list(resourceType, resourceId),
        });
      } else {
        queryClient.invalidateQueries({ queryKey: shareKeys.all });
      }
      toast.success('Share link revoked');
    },
    onError: () => {
      toast.error('Failed to revoke share link');
    },
  });
}
