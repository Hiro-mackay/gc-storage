import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { useTrashItems } from '../api/queries';
import {
  useRestoreFileMutation,
  usePermanentDeleteMutation,
  useEmptyTrashMutation,
} from '../api/mutations';
import { TrashList } from '../components/trash-list';
import { TrashToolbar } from '../components/trash-toolbar';
import { PermanentDeleteDialog } from '../components/permanent-delete-dialog';
import { BulkDeleteDialog } from '../components/bulk-delete-dialog';
import { EmptyTrashDialog } from '../components/empty-trash-dialog';

export function TrashPage() {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
  const [emptyConfirmOpen, setEmptyConfirmOpen] = useState(false);

  const { data, isLoading, error } = useTrashItems();
  const restoreMutation = useRestoreFileMutation();
  const permanentDeleteMutation = usePermanentDeleteMutation();
  const emptyTrashMutation = useEmptyTrashMutation();

  const items = data?.items ?? [];

  async function handleRestoreSelected() {
    const ids = Array.from(selectedIds);
    setSelectedIds(new Set());
    const results = await Promise.allSettled(
      ids.map((id) => restoreMutation.mutateAsync(id)),
    );
    const failed = results.filter((r) => r.status === 'rejected').length;
    if (failed > 0) {
      toast.error(`Failed to restore ${failed} item(s)`);
    }
  }

  async function handleDeleteSelectedConfirmed() {
    const ids = Array.from(selectedIds);
    setBulkDeleteOpen(false);
    setSelectedIds(new Set());
    const results = await Promise.allSettled(
      ids.map((id) => permanentDeleteMutation.mutateAsync(id)),
    );
    const failed = results.filter((r) => r.status === 'rejected').length;
    if (failed > 0) {
      toast.error(`Failed to delete ${failed} item(s)`);
    }
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <h1 className="text-2xl font-bold">Trash</h1>
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-4">Trash</h1>
        <p className="text-destructive">Failed to load trash.</p>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">Trash</h1>
        {items.length > 0 && (
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setEmptyConfirmOpen(true)}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Empty Trash
          </Button>
        )}
      </div>

      <p className="text-sm text-muted-foreground mb-4">
        Items in trash will be automatically deleted after 30 days.
      </p>

      {items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <Trash2 className="h-12 w-12 mb-4" />
          <p>Trash is empty</p>
        </div>
      ) : (
        <>
          <TrashToolbar
            selectedCount={selectedIds.size}
            onRestoreSelected={handleRestoreSelected}
            onDeleteSelected={() => setBulkDeleteOpen(true)}
            isRestoring={restoreMutation.isPending}
            isDeleting={permanentDeleteMutation.isPending}
          />
          <TrashList
            items={items}
            selectedIds={selectedIds}
            onSelectionChange={setSelectedIds}
            onRestore={(id) => restoreMutation.mutate(id)}
            onDelete={setDeleteTarget}
            isRestoring={restoreMutation.isPending}
          />
        </>
      )}

      <PermanentDeleteDialog
        target={deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        onConfirm={() => {
          if (deleteTarget) {
            permanentDeleteMutation.mutate(deleteTarget.id, {
              onSuccess: () => setDeleteTarget(null),
            });
          }
        }}
        isPending={permanentDeleteMutation.isPending}
      />

      <BulkDeleteDialog
        open={bulkDeleteOpen}
        onOpenChange={setBulkDeleteOpen}
        itemCount={selectedIds.size}
        onConfirm={handleDeleteSelectedConfirmed}
        isPending={permanentDeleteMutation.isPending}
      />

      <EmptyTrashDialog
        open={emptyConfirmOpen}
        onOpenChange={setEmptyConfirmOpen}
        itemCount={items.length}
        onConfirm={() => {
          emptyTrashMutation.mutate(undefined, {
            onSuccess: () => setEmptyConfirmOpen(false),
          });
        }}
        isPending={emptyTrashMutation.isPending}
      />
    </div>
  );
}
