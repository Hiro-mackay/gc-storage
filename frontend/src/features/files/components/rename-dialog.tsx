import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from 'sonner';

interface RenameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: { id: string; name: string; type: 'file' | 'folder' } | null;
}

export function RenameDialog({ open, onOpenChange, item }: RenameDialogProps) {
  const [name, setName] = useState('');
  const queryClient = useQueryClient();
  const [prevItem, setPrevItem] = useState(item);

  if (item !== prevItem) {
    setPrevItem(item);
    if (item) setName(item.name);
  }

  const mutation = useMutation({
    mutationFn: async (newName: string) => {
      if (!item) return;
      if (item.type === 'folder') {
        const { error } = await api.PATCH('/folders/{id}/rename', {
          params: { path: { id: item.id } },
          body: { name: newName },
        });
        if (error) throw error;
      } else {
        const { error } = await api.PATCH('/files/{id}/rename', {
          params: { path: { id: item.id } },
          body: { name: newName },
        });
        if (error) throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success(`${item?.type === 'folder' ? 'Folder' : 'File'} renamed`);
      onOpenChange(false);
    },
    onError: () => {
      toast.error('Failed to rename');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim() && name.trim() !== item?.name) {
      mutation.mutate(name.trim());
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="rename-input">Name</Label>
              <Input
                id="rename-input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={
                !name.trim() || name.trim() === item?.name || mutation.isPending
              }
            >
              {mutation.isPending ? 'Renaming...' : 'Rename'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
