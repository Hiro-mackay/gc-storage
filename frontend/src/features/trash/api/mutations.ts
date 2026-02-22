import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { trashKeys, folderKeys } from '@/lib/api/queries';
import { toast } from 'sonner';

export function useRestoreFileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/trash/files/{id}/restore', {
        params: { path: { id } },
        body: {},
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: trashKeys.all });
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success('File restored');
    },
    onError: () => {
      toast.error('Failed to restore file');
    },
  });
}

export function usePermanentDeleteMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.DELETE('/trash/files/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: trashKeys.all });
      toast.success('Permanently deleted');
    },
    onError: () => {
      toast.error('Failed to delete');
    },
  });
}

export function useEmptyTrashMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.DELETE('/trash');
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: trashKeys.all });
      toast.success('Trash emptied');
    },
    onError: () => {
      toast.error('Failed to empty trash');
    },
  });
}
