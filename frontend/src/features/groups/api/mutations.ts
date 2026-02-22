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

export function useCreateGroupMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; description?: string }) => {
      const { data, error } = await api.POST('/groups', {
        body: { name: input.name, description: input.description },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to create group');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Group created');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useUpdateGroupMutation(groupId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name?: string; description?: string }) => {
      const { data, error } = await api.PATCH('/groups/{id}', {
        params: { path: { id: groupId } },
        body: { name: input.name, description: input.description },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to update group');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.detail(groupId) });
      queryClient.invalidateQueries({ queryKey: groupKeys.lists() });
      toast.success('Group updated');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useDeleteGroupMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (groupId: string) => {
      const { error } = await api.DELETE('/groups/{id}', {
        params: { path: { id: groupId } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to delete group');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Group deleted');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useRemoveMemberMutation(groupId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userId: string) => {
      const { error } = await api.DELETE('/groups/{id}/members/{userId}', {
        params: { path: { id: groupId, userId } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to remove member');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.members(groupId) });
      toast.success('Member removed');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useChangeRoleMutation(groupId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      userId: string;
      role: 'viewer' | 'contributor';
    }) => {
      const { data, error } = await api.PATCH(
        '/groups/{id}/members/{userId}/role',
        {
          params: { path: { id: groupId, userId: input.userId } },
          body: { role: input.role },
        },
      );
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to change role');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.members(groupId) });
      queryClient.invalidateQueries({ queryKey: groupKeys.detail(groupId) });
      toast.success('Role updated');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useTransferOwnershipMutation(groupId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (newOwnerId: string) => {
      const { data, error } = await api.POST('/groups/{id}/transfer', {
        params: { path: { id: groupId } },
        body: { newOwnerId },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to transfer ownership');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Ownership transferred');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useLeaveGroupMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (groupId: string) => {
      const { error } = await api.POST('/groups/{id}/leave', {
        params: { path: { id: groupId } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to leave group');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Left group');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}
