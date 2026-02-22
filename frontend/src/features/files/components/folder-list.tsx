import { Link } from '@tanstack/react-router';
import { useSelectionStore } from '@/stores/selection-store';
import { FileContextMenu } from './file-context-menu';
import type { FileItemRef, FilePreviewRef } from '../types';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Checkbox } from '@/components/ui/checkbox';
import { cn, formatBytes } from '@/lib/utils';
import { Folder, FileIcon } from 'lucide-react';

interface FolderListProps {
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

export function FolderList({
  folders,
  files,
  onRename,
  onShare,
  onMove,
  onPreview,
}: FolderListProps) {
  const { toggle, clear, isSelected, selectAll } = useSelectionStore();

  const allIds = [
    ...folders.map((f) => f.id ?? ''),
    ...files.map((f) => f.id ?? ''),
  ].filter(Boolean);
  const allSelected = allIds.length > 0 && allIds.every((id) => isSelected(id));

  return (
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
              <TableCell className="text-muted-foreground">&mdash;</TableCell>
              <TableCell className="text-muted-foreground">
                {folder.updatedAt
                  ? new Date(folder.updatedAt).toLocaleDateString()
                  : '\u2014'}
              </TableCell>
            </TableRow>
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
                <button
                  type="button"
                  className="flex items-center gap-2 hover:underline text-left"
                  onClick={(e) => {
                    e.stopPropagation();
                    onPreview?.({
                      id: file.id ?? '',
                      name: file.name ?? '',
                      mimeType: file.mimeType ?? '',
                      size: file.size ?? 0,
                    });
                  }}
                >
                  <FileIcon className="h-4 w-4 text-gray-500" />
                  {file.name}
                </button>
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
  );
}
