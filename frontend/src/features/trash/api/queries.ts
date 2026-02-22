import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { trashKeys } from '@/lib/api/queries';

export interface TrashItem {
  id?: string;
  name?: string;
  mimeType?: string;
  originalPath?: string;
  originalFileId?: string;
  originalFolderId?: string;
  archivedAt?: string;
  expiresAt?: string;
  daysUntilExpiry?: number;
  size?: number;
}

export interface TrashListData {
  items?: TrashItem[];
  nextCursor?: string | null;
}

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
      } as never);
      if (error) throw error;
      return ((data as { data?: TrashListData })?.data ?? {
        items: [],
      }) as TrashListData;
    },
  });
}
