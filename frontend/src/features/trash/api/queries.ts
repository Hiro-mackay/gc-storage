import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { trashKeys } from '@/lib/api/queries';
import type { components } from '@/lib/api/schema';

export type TrashItem =
  components['schemas']['github_com_Hiro-mackay_gc-storage_backend_internal_interface_dto_response.TrashItemResponse'];

export function useTrashItems(limit?: number, cursor?: string) {
  return useQuery({
    queryKey: trashKeys.list(limit, cursor),
    queryFn: async () => {
      const { data, error } = await api.GET('/trash', {
        params: {
          query: {
            ...(limit != null ? { limit } : {}),
            ...(cursor ? { cursor } : {}),
          },
        },
      });
      if (error) throw error;
      return data?.data ?? { items: [] };
    },
  });
}
