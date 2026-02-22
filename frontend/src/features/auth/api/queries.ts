import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { authKeys } from '@/lib/api/queries';

export async function fetchCurrentUser() {
  const { data, error } = await api.GET('/me');
  if (error || !data?.data) return null;
  return data.data;
}

export function useMeQuery() {
  return useQuery({
    queryKey: authKeys.me(),
    queryFn: fetchCurrentUser,
  });
}
