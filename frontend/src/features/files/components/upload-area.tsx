import { useCallback, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import { useUploadStore } from '@/stores/upload-store';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import {
  Upload,
  X,
  FileIcon,
  CheckCircle2,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import { toast } from 'sonner';

interface UploadAreaProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  folderId: string | null;
}

export function UploadArea({ open, onOpenChange, folderId }: UploadAreaProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);
  const [stagedFiles, setStagedFiles] = useState<File[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const queryClient = useQueryClient();
  const { uploads, addUpload, updateProgress, setStatus, removeUpload } =
    useUploadStore();

  const uploadFile = useCallback(
    async (file: File) => {
      const id = crypto.randomUUID();
      addUpload({ id, fileName: file.name, fileSize: file.size });

      try {
        if (!folderId) {
          throw new Error('No folder selected');
        }

        // Initiate upload to get presigned URL
        const { data, error } = await api.POST('/files/upload', {
          body: {
            fileName: file.name,
            folderId: folderId,
            mimeType: file.type || 'application/octet-stream',
            size: file.size,
          },
        });

        if (error) {
          const msg =
            error &&
            typeof error === 'object' &&
            'error' in error &&
            (error as { error?: { message?: string } }).error?.message;
          throw new Error(msg || 'Failed to initiate upload');
        }

        const uploadData = data?.data;
        if (!uploadData?.uploadUrls?.[0]?.url) {
          throw new Error('No upload URL received');
        }

        // Upload directly to MinIO via presigned URL
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
    [folderId, addUpload, updateProgress, setStatus, queryClient],
  );

  const addFiles = useCallback((files: FileList | File[]) => {
    setStagedFiles((prev) => [...prev, ...Array.from(files)]);
  }, []);

  const removeStagedFile = useCallback((index: number) => {
    setStagedFiles((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const handleUpload = useCallback(async () => {
    if (stagedFiles.length === 0) return;
    setIsUploading(true);
    const filesToUpload = [...stagedFiles];
    setStagedFiles([]);
    await Promise.all(filesToUpload.map(uploadFile));
    setIsUploading(false);
  }, [stagedFiles, uploadFile]);

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragActive(false);
      if (e.dataTransfer.files?.length) {
        addFiles(e.dataTransfer.files);
      }
    },
    [addFiles],
  );

  const handleClose = (nextOpen: boolean) => {
    if (!nextOpen && !isUploading) {
      setStagedFiles([]);
    }
    onOpenChange(nextOpen);
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const uploadItems = Array.from(uploads.values());

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Upload Files</DialogTitle>
        </DialogHeader>
        <div
          className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
            dragActive
              ? 'border-primary bg-primary/5'
              : 'border-muted-foreground/25'
          }`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
        >
          <Upload className="h-10 w-10 mx-auto mb-3 text-muted-foreground" />
          <p className="text-sm text-muted-foreground mb-2">
            Drag and drop files here, or
          </p>
          <Button
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            disabled={isUploading}
          >
            Browse Files
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={(e) => {
              if (e.target.files?.length) {
                addFiles(e.target.files);
                e.target.value = '';
              }
            }}
          />
        </div>

        {/* Staged files (before upload) */}
        {stagedFiles.length > 0 && (
          <div className="mt-4 space-y-2 max-h-48 overflow-auto">
            {stagedFiles.map((file, index) => (
              <div
                key={`${file.name}-${index}`}
                className="flex items-center gap-3 rounded-md border p-2"
              >
                <FileIcon className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{file.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatSize(file.size)}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0"
                  onClick={() => removeStagedFile(index)}
                >
                  <X className="h-3 w-3" />
                </Button>
              </div>
            ))}
          </div>
        )}

        {/* Upload progress */}
        {uploadItems.length > 0 && (
          <div className="mt-4 space-y-2 max-h-48 overflow-auto">
            {uploadItems.map((item) => (
              <div
                key={item.id}
                className="flex items-center gap-3 rounded-md border p-2"
              >
                <FileIcon className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{item.fileName}</p>
                  <div className="h-1.5 bg-muted rounded-full mt-1">
                    <div
                      className={`h-full rounded-full transition-all ${
                        item.status === 'failed'
                          ? 'bg-destructive'
                          : item.status === 'completed'
                            ? 'bg-green-500'
                            : 'bg-primary'
                      }`}
                      style={{ width: `${item.progress}%` }}
                    />
                  </div>
                </div>
                <div className="shrink-0">
                  {item.status === 'completed' && (
                    <CheckCircle2 className="h-4 w-4 text-green-500" />
                  )}
                  {item.status === 'failed' && (
                    <AlertCircle className="h-4 w-4 text-destructive" />
                  )}
                  {(item.status === 'pending' ||
                    item.status === 'uploading') && (
                    <Loader2 className="h-4 w-4 animate-spin text-primary" />
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0"
                  onClick={() => removeUpload(item.id)}
                >
                  <X className="h-3 w-3" />
                </Button>
              </div>
            ))}
          </div>
        )}

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleClose(false)}
            disabled={isUploading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleUpload}
            disabled={stagedFiles.length === 0 || isUploading}
          >
            {isUploading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Uploading...
              </>
            ) : (
              <>
                <Upload className="h-4 w-4 mr-2" />
                Upload {stagedFiles.length > 0 && `(${stagedFiles.length})`}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
