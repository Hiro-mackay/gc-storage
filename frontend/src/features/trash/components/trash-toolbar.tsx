import { Button } from '@/components/ui/button';
import { RotateCcw, Trash2 } from 'lucide-react';

interface TrashToolbarProps {
  selectedCount: number;
  onRestoreSelected: () => void;
  onDeleteSelected: () => void;
  isRestoring: boolean;
  isDeleting: boolean;
}

export function TrashToolbar({
  selectedCount,
  onRestoreSelected,
  onDeleteSelected,
  isRestoring,
  isDeleting,
}: TrashToolbarProps) {
  if (selectedCount === 0) return null;

  return (
    <div className="flex items-center justify-between rounded-md border bg-muted/50 px-4 py-2 mb-4">
      <span className="text-sm font-medium">
        {selectedCount} item{selectedCount !== 1 ? 's' : ''} selected
      </span>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onRestoreSelected}
          disabled={isRestoring}
        >
          <RotateCcw className="h-4 w-4 mr-2" />
          Restore
        </Button>
        <Button
          variant="destructive"
          size="sm"
          onClick={onDeleteSelected}
          disabled={isDeleting}
        >
          <Trash2 className="h-4 w-4 mr-2" />
          Delete permanently
        </Button>
      </div>
    </div>
  );
}
