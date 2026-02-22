import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { formatBytes } from '@/lib/utils';
import { fileKeys } from '@/lib/api/queries';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Download, FileIcon } from 'lucide-react';
import type { FilePreviewRef } from '../types';

interface FilePreviewProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  file: FilePreviewRef | null;
}

export function FilePreview({ open, onOpenChange, file }: FilePreviewProps) {
  const { data: downloadUrl, isLoading: loading } = useQuery({
    queryKey: fileKeys.download(file?.id ?? ''),
    queryFn: async () => {
      const { data, error } = await api.GET('/files/{id}/download', {
        params: { path: { id: file!.id } },
      });
      if (error || !data?.data?.downloadUrl) return null;
      return data.data.downloadUrl;
    },
    enabled: open && !!file,
  });

  if (!file) return null;

  const isImage = file.mimeType.startsWith('image/');
  const isPdf = file.mimeType === 'application/pdf';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{file.name}</DialogTitle>
        </DialogHeader>
        <div className="flex items-center justify-center min-h-[200px]">
          {loading && (
            <p className="text-muted-foreground text-sm">Loading preview...</p>
          )}
          {!loading && downloadUrl && isImage && (
            <img
              src={downloadUrl}
              alt={file.name}
              className="max-w-full max-h-[70vh] object-contain"
            />
          )}
          {!loading && downloadUrl && isPdf && (
            <iframe
              src={downloadUrl}
              title={file.name}
              className="w-full h-[70vh]"
            />
          )}
          {!loading && (!downloadUrl || (!isImage && !isPdf)) && (
            <div className="flex flex-col items-center gap-3 py-8">
              <FileIcon className="h-16 w-16 text-muted-foreground" />
              <div className="text-center space-y-1">
                <p className="font-medium">{file.name}</p>
                <p className="text-sm text-muted-foreground">
                  {file.mimeType} - {formatBytes(file.size)}
                </p>
              </div>
              {downloadUrl && (
                <Button asChild variant="outline" size="sm">
                  <a href={downloadUrl} download={file.name}>
                    <Download className="h-4 w-4 mr-2" />
                    Download
                  </a>
                </Button>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
