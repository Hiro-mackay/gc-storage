import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import { useUploadStore } from '@/stores/upload-store';
import { toast } from 'sonner';

export function useFileUpload() {
  const queryClient = useQueryClient();
  const { addUpload, updateProgress, setStatus } = useUploadStore();

  const uploadFile = useCallback(
    async (file: File, folderId: string) => {
      const id = crypto.randomUUID();
      addUpload({ id, fileName: file.name, fileSize: file.size });

      try {
        // Initiate upload to get presigned URL
        const { data, error } = await api.POST('/files/upload', {
          body: {
            fileName: file.name,
            folderId,
            mimeType: file.type || 'application/octet-stream',
            size: file.size,
          },
        });

        if (error) {
          throw new Error(error.error?.message ?? 'Failed to initiate upload');
        }

        const uploadData = data?.data;
        if (!uploadData?.uploadUrls?.[0]?.url) {
          throw new Error('No upload URL received');
        }

        // Upload directly to storage via presigned URL
        const xhr = new XMLHttpRequest();
        xhr.open('PUT', uploadData.uploadUrls[0].url);
        xhr.setRequestHeader(
          'Content-Type',
          file.type || 'application/octet-stream',
        );

        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            updateProgress(id, Math.round((e.loaded / e.total) * 95));
          }
        };

        const etag = await new Promise<string>((resolve, reject) => {
          xhr.onload = () => {
            if (xhr.status >= 200 && xhr.status < 300) {
              resolve(xhr.getResponseHeader('ETag') ?? '');
            } else {
              reject(new Error(`Storage upload failed: ${xhr.status}`));
            }
          };
          xhr.onerror = () => reject(new Error('Network error during upload'));
          xhr.send(file);
        });

        // Complete upload
        const { error: completeError } = await api.POST(
          '/files/upload/complete',
          {
            body: {
              storageKey: uploadData.fileId ?? '',
              etag,
              size: file.size,
              minioVersionId: '',
            },
          },
        );

        if (completeError) {
          throw new Error('Failed to finalize upload');
        }

        updateProgress(id, 100);
        setStatus(id, 'completed');
        queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Upload failed';
        setStatus(id, 'failed', message);
        toast.error(`${file.name}: ${message}`);
      }
    },
    [addUpload, updateProgress, setStatus, queryClient],
  );

  return { uploadFile };
}
