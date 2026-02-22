import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { groupKeys } from '@/lib/api/queries';
import { toast } from 'sonner';

export function useInviteMemberMutation(groupId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      email: string;
      role?: 'viewer' | 'contributor';
    }) => {
      const { data, error } = await api.POST('/groups/{id}/invitations', {
        params: { path: { id: groupId } },
        body: { email: input.email, role: input.role ?? 'viewer' },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to send invitation');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: groupKeys.invitations(groupId),
      });
      toast.success('Invitation sent');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useAcceptInvitationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (token: string) => {
      const { data, error } = await api.POST('/invitations/{token}/accept', {
        params: { path: { token } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to accept invitation');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Invitation accepted');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useDeclineInvitationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (token: string) => {
      const { error } = await api.POST('/invitations/{token}/decline', {
        params: { path: { token } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to decline invitation');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.pending() });
      toast.success('Invitation declined');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useCancelInvitationMutation(groupId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (invitationId: string) => {
      const { error } = await api.DELETE(
        '/groups/{id}/invitations/{invitationId}',
        {
          params: { path: { id: groupId, invitationId } },
        },
      );
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to cancel invitation');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: groupKeys.invitations(groupId),
      });
      toast.success('Invitation cancelled');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}
