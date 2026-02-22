import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { shareKeys } from '@/lib/api/queries';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Copy, Link2, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

interface ShareDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: { id: string; name: string; type: 'file' | 'folder' } | null;
}

export function ShareDialog({ open, onOpenChange, item }: ShareDialogProps) {
  const [permission, setPermission] = useState<'read' | 'write'>('read');
  const queryClient = useQueryClient();

  const sharesQuery = useQuery({
    queryKey: shareKeys.list(item?.type ?? '', item?.id ?? ''),
    queryFn: async () => {
      if (!item) return [];
      if (item.type === 'file') {
        const { data, error } = await api.GET('/files/{id}/share', {
          params: { path: { id: item.id } },
        });
        if (error) throw error;
        return data?.data ?? [];
      } else {
        const { data, error } = await api.GET('/folders/{id}/share', {
          params: { path: { id: item.id } },
        });
        if (error) throw error;
        return data?.data ?? [];
      }
    },
    enabled: !!item && open,
  });

  const createMutation = useMutation({
    mutationFn: async () => {
      if (!item) return;
      if (item.type === 'file') {
        const { data, error } = await api.POST('/files/{id}/share', {
          params: { path: { id: item.id } },
          body: { permission },
        });
        if (error) throw error;
        return data?.data;
      } else {
        const { data, error } = await api.POST('/folders/{id}/share', {
          params: { path: { id: item.id } },
          body: { permission },
        });
        if (error) throw error;
        return data?.data;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: shareKeys.list(item?.type ?? '', item?.id ?? ''),
      });
      toast.success('Share link created');
    },
    onError: () => {
      toast.error('Failed to create share link');
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (linkId: string) => {
      const { error } = await api.DELETE('/share-links/{id}', {
        params: { path: { id: linkId } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: shareKeys.list(item?.type ?? '', item?.id ?? ''),
      });
      toast.success('Share link revoked');
    },
    onError: () => {
      toast.error('Failed to revoke share link');
    },
  });

  const copyLink = (url: string) => {
    navigator.clipboard.writeText(url);
    toast.success('Link copied to clipboard');
  };

  const shares = sharesQuery.data ?? [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Share "{item?.name}"</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="flex items-end gap-2">
            <div className="flex-1 space-y-2">
              <Label>Permission</Label>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant={permission === 'read' ? 'default' : 'outline'}
                  onClick={() => setPermission('read')}
                >
                  View only
                </Button>
                <Button
                  size="sm"
                  variant={permission === 'write' ? 'default' : 'outline'}
                  onClick={() => setPermission('write')}
                >
                  Can edit
                </Button>
              </div>
            </div>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={createMutation.isPending}
            >
              <Link2 className="h-4 w-4 mr-2" />
              {createMutation.isPending ? 'Creating...' : 'Create Link'}
            </Button>
          </div>

          {shares.length > 0 && (
            <div className="space-y-2">
              <Label>Active Links</Label>
              {shares.map((link) => (
                <div
                  key={link.id}
                  className="flex items-center gap-2 rounded-md border p-2"
                >
                  <Link2 className="h-4 w-4 text-muted-foreground shrink-0" />
                  <div className="flex-1 min-w-0">
                    <Input
                      readOnly
                      value={link.url ?? ''}
                      className="h-8 text-xs"
                    />
                  </div>
                  <Badge variant="secondary" className="shrink-0">
                    {link.permission}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    onClick={() => copyLink(link.url ?? '')}
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0 text-destructive"
                    onClick={() => link.id && revokeMutation.mutate(link.id)}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
