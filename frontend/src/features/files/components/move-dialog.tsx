import { useState } from 'react';
import { useFolderContents } from '../api/queries';
import { useMoveFileMutation, useMoveFolderMutation } from '../api/mutations';
import type { FileItemRef } from '../types';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { ChevronRight, Folder } from 'lucide-react';

interface MoveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: FileItemRef | null;
}

export function MoveDialog({ open, onOpenChange, item }: MoveDialogProps) {
  const [selectedFolderId, setSelectedFolderId] = useState<string | null>(null);
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(
    new Set(['root']),
  );

  const moveFile = useMoveFileMutation();
  const moveFolder = useMoveFolderMutation();

  const handleMove = () => {
    if (!item || !selectedFolderId) return;

    if (item.type === 'file') {
      moveFile.mutate(
        { id: item.id, newFolderId: selectedFolderId },
        { onSuccess: () => onOpenChange(false) },
      );
    } else {
      moveFolder.mutate(
        { id: item.id, newParentId: selectedFolderId },
        { onSuccess: () => onOpenChange(false) },
      );
    }
  };

  const isMoving = moveFile.isPending || moveFolder.isPending;

  const toggleExpand = (folderId: string) => {
    setExpandedFolders((prev) => {
      const next = new Set(prev);
      if (next.has(folderId)) {
        next.delete(folderId);
      } else {
        next.add(folderId);
      }
      return next;
    });
  };

  const movedItemId = item?.type === 'folder' ? item.id : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Move &ldquo;{item?.name}&rdquo;</DialogTitle>
        </DialogHeader>
        <ScrollArea className="h-[300px] rounded-md border p-2">
          <FolderTreeNode
            folderId={null}
            label="My Files"
            depth={0}
            selectedFolderId={selectedFolderId}
            expandedFolders={expandedFolders}
            onSelect={setSelectedFolderId}
            onToggleExpand={toggleExpand}
            movedItemId={movedItemId}
            subtreeDisabled={false}
          />
        </ScrollArea>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleMove} disabled={!selectedFolderId || isMoving}>
            {isMoving ? 'Moving...' : 'Move here'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface FolderTreeNodeProps {
  folderId: string | null;
  label: string;
  depth: number;
  selectedFolderId: string | null;
  expandedFolders: Set<string>;
  onSelect: (id: string) => void;
  onToggleExpand: (id: string) => void;
  movedItemId: string | null;
  subtreeDisabled: boolean;
}

function FolderTreeNode({
  folderId,
  label,
  depth,
  selectedFolderId,
  expandedFolders,
  onSelect,
  onToggleExpand,
  movedItemId,
  subtreeDisabled,
}: FolderTreeNodeProps) {
  const nodeId = folderId ?? 'root';
  const isExpanded = expandedFolders.has(nodeId);

  // Always fetch for root to resolve the real folder UUID
  const shouldFetch = folderId === null || isExpanded;
  const { data } = useFolderContents(shouldFetch ? folderId : null);
  const childFolders = data?.folders ?? [];

  // Resolve actual folder ID (root alias -> real UUID)
  const resolvedId =
    folderId === null && data?.folder?.id ? data.folder.id : nodeId;

  // Disable this node if it's the folder being moved, or inside a disabled subtree
  const selfDisabled = movedItemId === resolvedId;
  const nodeDisabled = subtreeDisabled || selfDisabled;

  return (
    <div>
      <button
        type="button"
        className={cn(
          'flex w-full items-center gap-1 rounded-md px-2 py-1.5 text-sm hover:bg-accent',
          selectedFolderId === resolvedId && 'bg-accent font-medium',
          nodeDisabled && 'opacity-50 cursor-not-allowed',
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={() => {
          if (nodeDisabled) return;
          onSelect(resolvedId);
          if (!isExpanded) onToggleExpand(nodeId);
        }}
        disabled={nodeDisabled}
      >
        <ChevronRight
          className={cn(
            'h-4 w-4 shrink-0 transition-transform',
            isExpanded && 'rotate-90',
          )}
          onClick={(e) => {
            e.stopPropagation();
            if (!nodeDisabled) onToggleExpand(nodeId);
          }}
        />
        <Folder className="h-4 w-4 shrink-0 text-blue-500" />
        <span className="truncate">{label}</span>
      </button>
      {isExpanded &&
        childFolders.map((child) => (
          <FolderTreeNode
            key={child.id}
            folderId={child.id ?? ''}
            label={child.name ?? ''}
            depth={depth + 1}
            selectedFolderId={selectedFolderId}
            expandedFolders={expandedFolders}
            onSelect={onSelect}
            onToggleExpand={onToggleExpand}
            movedItemId={movedItemId}
            subtreeDisabled={nodeDisabled}
          />
        ))}
    </div>
  );
}
