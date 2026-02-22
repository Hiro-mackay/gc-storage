import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import { toast } from 'sonner';

export function useCreateFolderMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; parentId?: string }) => {
      const { data, error } = await api.POST('/folders', {
        body: input,
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to create folder');
      }
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success('Folder created');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useRenameMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      id: string;
      name: string;
      type: 'file' | 'folder';
    }) => {
      if (input.type === 'folder') {
        const { error } = await api.PATCH('/folders/{id}/rename', {
          params: { path: { id: input.id } },
          body: { name: input.name },
        });
        if (error) {
          throw new Error(error.error?.message ?? 'Failed to rename folder');
        }
      } else {
        const { error } = await api.PATCH('/files/{id}/rename', {
          params: { path: { id: input.id } },
          body: { name: input.name },
        });
        if (error) {
          throw new Error(error.error?.message ?? 'Failed to rename file');
        }
      }
    },
    onSuccess: (_data, input) => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success(`${input.type === 'folder' ? 'Folder' : 'File'} renamed`);
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useTrashMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { id: string; type: 'file' | 'folder' }) => {
      if (input.type === 'file') {
        const { error } = await api.POST('/files/{id}/trash', {
          params: { path: { id: input.id } },
        });
        if (error) {
          throw new Error(error.error?.message ?? 'Failed to trash file');
        }
      } else {
        const { error } = await api.DELETE('/folders/{id}', {
          params: { path: { id: input.id } },
        });
        if (error) {
          throw new Error(error.error?.message ?? 'Failed to delete folder');
        }
      }
    },
    onSuccess: (_data, input) => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success(
        `${input.type === 'folder' ? 'Folder' : 'File'} moved to trash`,
      );
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useMoveFileMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { id: string; newFolderId: string }) => {
      const { error } = await api.PATCH('/files/{id}/move', {
        params: { path: { id: input.id } },
        body: { newFolderId: input.newFolderId },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to move file');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success('File moved');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useMoveFolderMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { id: string; newParentId: string }) => {
      const { error } = await api.PATCH('/folders/{id}/move', {
        params: { path: { id: input.id } },
        body: { newParentId: input.newParentId },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to move folder');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success('Folder moved');
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
}

export function useAbortUploadMutation() {
  return useMutation({
    mutationFn: async (sessionId: string) => {
      const { error } = await api.DELETE('/files/upload/{sessionId}', {
        params: { path: { sessionId } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Failed to abort upload');
      }
    },
  });
}
