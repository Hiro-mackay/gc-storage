import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Trash2, RotateCcw, FileIcon } from 'lucide-react';
import type { TrashItem } from '../api/queries';

interface TrashListProps {
  items: TrashItem[];
  selectedIds: Set<string>;
  onSelectionChange: (ids: Set<string>) => void;
  onRestore: (id: string) => void;
  onDelete: (item: { id: string; name: string }) => void;
  isRestoring: boolean;
}

export function TrashList({
  items,
  selectedIds,
  onSelectionChange,
  onRestore,
  onDelete,
  isRestoring,
}: TrashListProps) {
  const allSelected =
    items.length > 0 && items.every((i) => i.id && selectedIds.has(i.id));
  const someSelected = items.some((i) => i.id && selectedIds.has(i.id));

  function handleSelectAll(checked: boolean) {
    if (checked) {
      const allIds = new Set(
        items.map((i) => i.id).filter(Boolean) as string[],
      );
      onSelectionChange(allIds);
    } else {
      onSelectionChange(new Set());
    }
  }

  function handleSelectItem(id: string, checked: boolean) {
    const next = new Set(selectedIds);
    if (checked) {
      next.add(id);
    } else {
      next.delete(id);
    }
    onSelectionChange(next);
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[40px]">
            <Checkbox
              checked={
                allSelected ? true : someSelected ? 'indeterminate' : false
              }
              onCheckedChange={(checked) => handleSelectAll(checked === true)}
              aria-label="Select all"
            />
          </TableHead>
          <TableHead className="w-[40%]">Name</TableHead>
          <TableHead>Original Location</TableHead>
          <TableHead>Deleted</TableHead>
          <TableHead>Expires</TableHead>
          <TableHead className="w-[140px]">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>
              <Checkbox
                checked={item.id ? selectedIds.has(item.id) : false}
                onCheckedChange={(checked) =>
                  item.id && handleSelectItem(item.id, checked === true)
                }
                aria-label={`Select ${item.name}`}
              />
            </TableCell>
            <TableCell>
              <div className="flex items-center gap-2">
                <FileIcon className="h-4 w-4 text-gray-500" />
                {item.name}
              </div>
            </TableCell>
            <TableCell className="text-muted-foreground text-sm">
              {item.originalPath ?? '/'}
            </TableCell>
            <TableCell className="text-muted-foreground text-sm">
              {item.archivedAt
                ? new Date(item.archivedAt).toLocaleDateString()
                : '\u2014'}
            </TableCell>
            <TableCell>
              {item.daysUntilExpiry != null && (
                <Badge
                  variant={
                    item.daysUntilExpiry <= 3 ? 'destructive' : 'secondary'
                  }
                >
                  {item.daysUntilExpiry}d left
                </Badge>
              )}
            </TableCell>
            <TableCell>
              <div className="flex gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => item.id && onRestore(item.id)}
                  disabled={isRestoring}
                  title="Restore"
                >
                  <RotateCcw className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive"
                  onClick={() =>
                    item.id &&
                    item.name &&
                    onDelete({ id: item.id, name: item.name })
                  }
                  title="Permanently Delete"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
