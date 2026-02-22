import { Link } from '@tanstack/react-router';
import { useSelectionStore } from '@/stores/selection-store';
import { FileContextMenu } from './file-context-menu';
import type { FileItemRef, FilePreviewRef } from '../types';
import { cn, formatBytes } from '@/lib/utils';
import { Folder, FileIcon } from 'lucide-react';

interface FolderGridProps {
  folders: Array<{ id?: string; name?: string; updatedAt?: string }>;
  files: Array<{
    id?: string;
    name?: string;
    size?: number;
    updatedAt?: string;
    mimeType?: string;
  }>;
  onRename: (item: FileItemRef) => void;
  onShare: (item: FileItemRef) => void;
  onMove: (item: FileItemRef) => void;
  onPreview?: (file: FilePreviewRef) => void;
}

export function FolderGrid({
  folders,
  files,
  onRename,
  onShare,
  onMove,
  onPreview,
}: FolderGridProps) {
  const { toggle, isSelected } = useSelectionStore();

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
      {folders.map((folder) => (
        <FileContextMenu
          key={folder.id}
          item={{
            id: folder.id ?? '',
            name: folder.name ?? '',
            type: 'folder',
          }}
          onRename={() =>
            onRename({
              id: folder.id ?? '',
              name: folder.name ?? '',
              type: 'folder',
            })
          }
          onShare={() =>
            onShare({
              id: folder.id ?? '',
              name: folder.name ?? '',
              type: 'folder',
            })
          }
          onMove={() =>
            onMove({
              id: folder.id ?? '',
              name: folder.name ?? '',
              type: 'folder',
            })
          }
        >
          <Link
            to="/files/$folderId"
            params={{ folderId: folder.id ?? '' }}
            className={cn(
              'flex flex-col items-center gap-2 rounded-lg border p-4 transition-colors hover:bg-accent',
              isSelected(folder.id ?? '') && 'bg-accent ring-2 ring-primary',
            )}
            onClick={(e) => {
              if (e.ctrlKey || e.metaKey) {
                e.preventDefault();
                toggle(folder.id ?? '');
              }
            }}
          >
            <Folder className="h-10 w-10 text-blue-500" />
            <span className="text-sm font-medium truncate w-full text-center">
              {folder.name}
            </span>
          </Link>
        </FileContextMenu>
      ))}
      {files.map((file) => (
        <FileContextMenu
          key={file.id}
          item={{
            id: file.id ?? '',
            name: file.name ?? '',
            type: 'file',
          }}
          onRename={() =>
            onRename({
              id: file.id ?? '',
              name: file.name ?? '',
              type: 'file',
            })
          }
          onShare={() =>
            onShare({
              id: file.id ?? '',
              name: file.name ?? '',
              type: 'file',
            })
          }
          onMove={() =>
            onMove({
              id: file.id ?? '',
              name: file.name ?? '',
              type: 'file',
            })
          }
        >
          <button
            type="button"
            className={cn(
              'flex flex-col items-center gap-2 rounded-lg border p-4 transition-colors hover:bg-accent w-full',
              isSelected(file.id ?? '') && 'bg-accent ring-2 ring-primary',
            )}
            onClick={(e) => {
              if (e.ctrlKey || e.metaKey) {
                toggle(file.id ?? '');
              } else {
                onPreview?.({
                  id: file.id ?? '',
                  name: file.name ?? '',
                  mimeType: file.mimeType ?? '',
                  size: file.size ?? 0,
                });
              }
            }}
          >
            <FileIcon className="h-10 w-10 text-gray-500" />
            <span className="text-sm font-medium truncate w-full text-center">
              {file.name}
            </span>
            {file.size != null && (
              <span className="text-xs text-muted-foreground">
                {formatBytes(file.size)}
              </span>
            )}
          </button>
        </FileContextMenu>
      ))}
    </div>
  );
}
