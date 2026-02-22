import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { profileKeys } from '@/lib/api/queries';
import { toast } from 'sonner';

export function useUpdateProfileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: {
      bio?: string;
      locale?: string;
      timezone?: string;
      theme?: 'system' | 'light' | 'dark';
      notification_preferences?: {
        email_enabled?: boolean;
        push_enabled?: boolean;
      };
    }) => {
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
      const { error } = await api.PUT('/me', { body });
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
