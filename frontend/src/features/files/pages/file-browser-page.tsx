import { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from '@tanstack/react-router';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import { useSelectionStore } from '@/stores/selection-store';
import { useUIStore } from '@/stores/ui-store';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { FileBreadcrumb } from '../components/file-breadcrumb';
import { FileToolbar } from '../components/file-toolbar';
import { CreateFolderDialog } from '../components/create-folder-dialog';
import { UploadArea } from '../components/upload-area';
import { RenameDialog } from '../components/rename-dialog';
import { ShareDialog } from '../components/share-dialog';
import { FileContextMenu } from '../components/file-context-menu';
import { Folder, FileIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

export function FileBrowserPage() {
  const params = useParams({ strict: false });
  const folderId = (params as { folderId?: string }).folderId ?? null;

  const { toggle, clear, isSelected, selectAll } = useSelectionStore();
  const { sortBy, sortOrder } = useUIStore();

  const [createFolderOpen, setCreateFolderOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [renameItem, setRenameItem] = useState<{
    id: string;
    name: string;
    type: 'file' | 'folder';
  } | null>(null);
  const [shareItem, setShareItem] = useState<{
    id: string;
    name: string;
    type: 'file' | 'folder';
  } | null>(null);

  const { data, isLoading, error, refetch } = useQuery({
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

  // Use actual folder ID from API response (resolves 'root' to personal folder UUID)
  const effectiveFolderId = data?.folder?.id ?? folderId;

  const folders = data?.folders ?? [];
  const files = data?.files ?? [];

  // Sort items
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

  const allIds = [
    ...sortedFolders.map((f) => f.id ?? ''),
    ...sortedFiles.map((f) => f.id ?? ''),
  ].filter(Boolean);
  const isEmpty = folders.length === 0 && files.length === 0;
  const allSelected = allIds.length > 0 && allIds.every((id) => isSelected(id));

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
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[40px]">
                <Checkbox
                  checked={allSelected}
                  onCheckedChange={(checked) => {
                    if (checked) {
                      selectAll(allIds);
                    } else {
                      clear();
                    }
                  }}
                />
              </TableHead>
              <TableHead className="w-[50%]">Name</TableHead>
              <TableHead>Size</TableHead>
              <TableHead>Modified</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedFolders.map((folder) => (
              <FileContextMenu
                key={folder.id}
                item={{
                  id: folder.id ?? '',
                  name: folder.name ?? '',
                  type: 'folder',
                }}
                onRename={() =>
                  setRenameItem({
                    id: folder.id ?? '',
                    name: folder.name ?? '',
                    type: 'folder',
                  })
                }
                onShare={() =>
                  setShareItem({
                    id: folder.id ?? '',
                    name: folder.name ?? '',
                    type: 'folder',
                  })
                }
              >
                <TableRow
                  className={cn(
                    'cursor-pointer',
                    isSelected(folder.id ?? '') && 'bg-accent',
                  )}
                  onClick={(e) => {
                    if (e.ctrlKey || e.metaKey) {
                      toggle(folder.id ?? '');
                    }
                  }}
                >
                  <TableCell>
                    <Checkbox
                      checked={isSelected(folder.id ?? '')}
                      onCheckedChange={() => toggle(folder.id ?? '')}
                      onClick={(e) => e.stopPropagation()}
                    />
                  </TableCell>
                  <TableCell>
                    <Link
                      to="/files/$folderId"
                      params={{ folderId: folder.id ?? '' }}
                      className="flex items-center gap-2 hover:underline"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <Folder className="h-4 w-4 text-blue-500" />
                      {folder.name}
                    </Link>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    &mdash;
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {folder.updatedAt
                      ? new Date(folder.updatedAt).toLocaleDateString()
                      : '\u2014'}
                  </TableCell>
                </TableRow>
              </FileContextMenu>
            ))}
            {sortedFiles.map((file) => (
              <FileContextMenu
                key={file.id}
                item={{
                  id: file.id ?? '',
                  name: file.name ?? '',
                  type: 'file',
                }}
                onRename={() =>
                  setRenameItem({
                    id: file.id ?? '',
                    name: file.name ?? '',
                    type: 'file',
                  })
                }
                onShare={() =>
                  setShareItem({
                    id: file.id ?? '',
                    name: file.name ?? '',
                    type: 'file',
                  })
                }
              >
                <TableRow
                  className={cn(
                    'cursor-pointer',
                    isSelected(file.id ?? '') && 'bg-accent',
                  )}
                  onClick={(e) => {
                    if (e.ctrlKey || e.metaKey) {
                      toggle(file.id ?? '');
                    }
                  }}
                >
                  <TableCell>
                    <Checkbox
                      checked={isSelected(file.id ?? '')}
                      onCheckedChange={() => toggle(file.id ?? '')}
                      onClick={(e) => e.stopPropagation()}
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <FileIcon className="h-4 w-4 text-gray-500" />
                      {file.name}
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {file.size ? formatBytes(file.size) : '\u2014'}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {file.updatedAt
                      ? new Date(file.updatedAt).toLocaleDateString()
                      : '\u2014'}
                  </TableCell>
                </TableRow>
              </FileContextMenu>
            ))}
          </TableBody>
        </Table>
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
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}
