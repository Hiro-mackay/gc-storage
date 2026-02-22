import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { fileKeys, folderKeys, uploadKeys } from '@/lib/api/queries';

export function useFolderContents(folderId: string | null) {
  return useQuery({
    queryKey: folderKeys.contents(folderId),
    queryFn: async () => {
      const id = folderId ?? 'root';
      const { data, error } = await api.GET('/folders/{id}/contents', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data?.data;
    },
  });
}

export function useFolderAncestors(folderId: string | null) {
  return useQuery({
    queryKey: folderKeys.ancestors(folderId ?? ''),
    queryFn: async () => {
      if (!folderId) return [];
      const { data, error } = await api.GET('/folders/{id}/ancestors', {
        params: { path: { id: folderId } },
      });
      if (error) throw error;
      return data?.data?.items ?? [];
    },
    enabled: !!folderId,
  });
}

export function useFileVersions(fileId: string) {
  return useQuery({
    queryKey: fileKeys.versions(fileId),
    queryFn: async () => {
      const { data, error } = await api.GET('/files/{id}/versions', {
        params: { path: { id: fileId } },
      });
      if (error) throw error;
      return data?.data?.versions ?? [];
    },
  });
}

export function useUploadStatus(sessionId: string) {
  return useQuery({
    queryKey: uploadKeys.status(sessionId),
    queryFn: async () => {
      const { data, error } = await api.GET('/files/upload/{sessionId}', {
        params: { path: { sessionId } },
      });
      if (error) throw error;
      return data?.data;
    },
    refetchInterval: 1000,
    enabled: !!sessionId,
  });
}
