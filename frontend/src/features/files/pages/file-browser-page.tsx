import { useState, useCallback } from 'react';
import { useParams } from '@tanstack/react-router';
import { useFolderContents } from '../api/queries';
import { useUIStore } from '@/stores/ui-store';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { FileBreadcrumb } from '../components/file-breadcrumb';
import { FileToolbar } from '../components/file-toolbar';
import { CreateFolderDialog } from '../components/create-folder-dialog';
import { UploadArea } from '../components/upload-area';
import { RenameDialog } from '../components/rename-dialog';
import { ShareDialog } from '../components/share-dialog';
import { MoveDialog } from '../components/move-dialog';
import { FolderList } from '../components/folder-list';
import { FolderGrid } from '../components/folder-grid';
import { FilePreview } from '../components/file-preview';
import { FileIcon } from 'lucide-react';
import type { FileItemRef, FilePreviewRef } from '../types';

export function FileBrowserPage() {
  const params = useParams({ strict: false });
  const folderId = (params as { folderId?: string }).folderId ?? null;

  const { sortBy, sortOrder, viewMode } = useUIStore();

  const [createFolderOpen, setCreateFolderOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [renameItem, setRenameItem] = useState<FileItemRef | null>(null);
  const [shareItem, setShareItem] = useState<FileItemRef | null>(null);
  const [moveItem, setMoveItem] = useState<FileItemRef | null>(null);
  const [previewFile, setPreviewFile] = useState<FilePreviewRef | null>(null);

  const { data, isLoading, error, refetch } = useFolderContents(folderId);

  const effectiveFolderId = data?.folder?.id ?? folderId;

  const folders = data?.folders ?? [];
  const files = data?.files ?? [];

  const sortedFolders = [...folders].sort((a, b) => {
    const dir = sortOrder === 'asc' ? 1 : -1;
    if (sortBy === 'name')
      return dir * (a.name ?? '').localeCompare(b.name ?? '');
    if (sortBy === 'updatedAt')
      return dir * (a.updatedAt ?? '').localeCompare(b.updatedAt ?? '');
    return 0;
  });

  const sortedFiles = [...files].sort((a, b) => {
    const dir = sortOrder === 'asc' ? 1 : -1;
    if (sortBy === 'name')
      return dir * (a.name ?? '').localeCompare(b.name ?? '');
    if (sortBy === 'updatedAt')
      return dir * (a.updatedAt ?? '').localeCompare(b.updatedAt ?? '');
    if (sortBy === 'size') return dir * ((a.size ?? 0) - (b.size ?? 0));
    return 0;
  });

  const isEmpty = folders.length === 0 && files.length === 0;

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    if (e.dataTransfer.files?.length) {
      setUploadOpen(true);
    }
  }, []);

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-10 w-full" />
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center p-6 py-16 text-muted-foreground">
        <FileIcon className="h-12 w-12 mb-4" />
        <p>Could not load folder contents</p>
        <p className="text-sm mt-1">
          Please check your connection and try again
        </p>
        <Button
          variant="outline"
          size="sm"
          className="mt-4"
          onClick={() => refetch()}
        >
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div
      className="p-6 space-y-4"
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      <FileBreadcrumb folderId={folderId} folderName={data?.folder?.name} />
      <FileToolbar
        onUpload={() => setUploadOpen(true)}
        onCreateFolder={() => setCreateFolderOpen(true)}
      />

      {isEmpty ? (
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <FileIcon className="h-12 w-12 mb-4" />
          <p>This folder is empty</p>
          <p className="text-sm mt-1">
            Upload files or create a folder to get started
          </p>
        </div>
      ) : viewMode === 'grid' ? (
        <FolderGrid
          folders={sortedFolders}
          files={sortedFiles}
          onRename={setRenameItem}
          onShare={setShareItem}
          onMove={setMoveItem}
          onPreview={setPreviewFile}
        />
      ) : (
        <FolderList
          folders={sortedFolders}
          files={sortedFiles}
          onRename={setRenameItem}
          onShare={setShareItem}
          onMove={setMoveItem}
          onPreview={setPreviewFile}
        />
      )}

      <CreateFolderDialog
        open={createFolderOpen}
        onOpenChange={setCreateFolderOpen}
        parentId={effectiveFolderId ?? null}
      />
      <UploadArea
        open={uploadOpen}
        onOpenChange={setUploadOpen}
        folderId={effectiveFolderId ?? null}
      />
      <RenameDialog
        open={!!renameItem}
        onOpenChange={(open) => {
          if (!open) setRenameItem(null);
        }}
        item={renameItem}
      />
      <ShareDialog
        open={!!shareItem}
        onOpenChange={(open) => {
          if (!open) setShareItem(null);
        }}
        item={shareItem}
      />
      <MoveDialog
        open={!!moveItem}
        onOpenChange={(open) => {
          if (!open) setMoveItem(null);
        }}
        item={moveItem}
      />
      <FilePreview
        open={!!previewFile}
        onOpenChange={(open) => {
          if (!open) setPreviewFile(null);
        }}
        file={previewFile}
      />
    </div>
  );
}
