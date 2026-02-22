import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { shareKeys } from '@/lib/api/queries';

export function useSharedResource(token: string) {
  return useQuery({
    queryKey: ['shared', token],
    queryFn: async () => {
      const { data, error } = await api.GET('/share/{token}', {
        params: { path: { token } },
      });
      if (error) throw error;
      return data?.data ?? null;
    },
    enabled: !!token,
    retry: false,
  });
}

export function useShareLinkHistory(shareLinkId: string) {
  return useQuery({
    queryKey: shareKeys.history(shareLinkId),
    queryFn: async () => {
      const { data, error } = await api.GET(
        '/share-links/{id}/history' as never,
        {
          params: { path: { id: shareLinkId } },
        } as never,
      );
      if (error) throw error;
      return (data as { data?: unknown })?.data ?? [];
    },
    enabled: !!shareLinkId,
  });
}
