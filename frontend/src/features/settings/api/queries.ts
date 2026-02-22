import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { profileKeys } from '@/lib/api/queries';

export function useProfile() {
  return useQuery({
    queryKey: profileKeys.all,
    queryFn: async () => {
      const { data, error } = await api.GET('/me/profile');
      if (error) throw error;
      return data?.data ?? null;
    },
  });
}
