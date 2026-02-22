import { api } from '@/lib/api/client';

export async function fetchCurrentUser() {
  const { data, error } = await api.GET('/me');
  if (error || !data?.data) return null;
  return data.data;
}
