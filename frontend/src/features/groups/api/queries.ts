import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { groupKeys } from '@/lib/api/queries';

export function useGroupInvitations(groupId: string) {
  return useQuery({
    queryKey: groupKeys.invitations(groupId),
    queryFn: async () => {
      const { data, error } = await api.GET('/groups/{id}/invitations', {
        params: { path: { id: groupId } },
      });
      if (error) throw error;
      return data?.data ?? [];
    },
  });
}

export function usePendingInvitations() {
  return useQuery({
    queryKey: groupKeys.pending(),
    queryFn: async () => {
      const { data, error } = await api.GET('/invitations/pending');
      if (error) throw error;
      return data?.data ?? [];
    },
  });
}
