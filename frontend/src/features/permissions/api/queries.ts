import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { permissionKeys } from '@/lib/api/queries';

export function useResourcePermissions(
  resourceType: 'file' | 'folder',
  resourceId: string,
) {
  return useQuery({
    queryKey: permissionKeys.resource(resourceType, resourceId),
    queryFn: async () => {
      if (resourceType === 'file') {
        const { data, error } = await api.GET('/files/{id}/permissions', {
          params: { path: { id: resourceId } },
        });
        if (error) throw error;
        return data?.data ?? [];
      } else {
        const { data, error } = await api.GET('/folders/{id}/permissions', {
          params: { path: { id: resourceId } },
        });
        if (error) throw error;
        return data?.data ?? [];
      }
    },
    enabled: !!resourceId,
  });
}
