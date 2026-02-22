import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';

export function useFolderContents(folderId: string | null) {
  return useQuery({
    queryKey: folderKeys.contents(folderId),
    queryFn: async () => {
      const id = folderId ?? 'root';
      const { data, error } = await api.GET('/folders/{id}/contents', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data?.data;
    },
  });
}

export function useFolderAncestors(folderId: string | null) {
  return useQuery({
    queryKey: folderKeys.ancestors(folderId ?? ''),
    queryFn: async () => {
      if (!folderId) return [];
      const { data, error } = await api.GET('/folders/{id}/ancestors', {
        params: { path: { id: folderId } },
      });
      if (error) throw error;
      return data?.data?.items ?? [];
    },
    enabled: !!folderId,
  });
}
