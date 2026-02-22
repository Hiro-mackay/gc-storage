import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { profileKeys } from '@/lib/api/queries';
import { toast } from 'sonner';
import type { components } from '@/lib/api/schema';

type UpdateProfileBody =
  components['schemas']['github_com_Hiro-mackay_gc-storage_backend_internal_interface_dto_request.UpdateProfileRequest'];

export function useUpdateProfileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: UpdateProfileBody) => {
      const { error } = await api.PUT('/me/profile', { body });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profileKeys.all });
    },
    onError: () => {
      toast.error('Failed to update profile');
    },
  });
}

export function useUpdateUserMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: { name?: string }) => {
      const { error } = await (
        api as unknown as {
          PUT: (
            path: string,
            opts: { body: { name?: string } },
          ) => Promise<{ error: unknown }>;
        }
      ).PUT('/me', { body });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profileKeys.all });
    },
    onError: () => {
      toast.error('Failed to update user');
    },
  });
}
