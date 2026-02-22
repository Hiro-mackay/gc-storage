import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { permissionKeys } from '@/lib/api/queries';
import { toast } from 'sonner';

type GrantInput = {
  granteeType: 'user' | 'group';
  granteeId: string;
  role: 'viewer' | 'contributor' | 'content_manager';
};

export function useGrantRoleMutation(
  resourceType: 'file' | 'folder',
  resourceId: string,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: GrantInput) => {
      if (resourceType === 'file') {
        const { data, error } = await api.POST('/files/{id}/permissions', {
          params: { path: { id: resourceId } },
          body: input,
        });
        if (error) {
          throw new Error(error.error?.message ?? 'Failed to grant permission');
        }
        return data!;
      } else {
        const { data, error } = await api.POST('/folders/{id}/permissions', {
          params: { path: { id: resourceId } },
          body: input,
        });
        if (error) {
          throw new Error(error.error?.message ?? 'Failed to grant permission');
        }
        return data!;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: permissionKeys.resource(resourceType, resourceId),
      });
      toast.success('Permission granted');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useRevokeGrantMutation(
  resourceType: 'file' | 'folder',
  resourceId: string,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (permissionId: string) => {
      const { error } = await api.DELETE('/permissions/{id}', {
        params: { path: { id: permissionId } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to revoke permission');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: permissionKeys.resource(resourceType, resourceId),
      });
      toast.success('Permission revoked');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}
