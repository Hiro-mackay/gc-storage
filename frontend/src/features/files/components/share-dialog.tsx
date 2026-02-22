import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { shareKeys } from '@/lib/api/queries';
import {
  useCreateShareLinkMutation,
  useRevokeShareLinkMutation,
} from '@/features/sharing/api/mutations';
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
  const [password, setPassword] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [maxAccessCount, setMaxAccessCount] = useState('');

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

  const createMutation = useCreateShareLinkMutation(
    item?.type ?? 'file',
    item?.id ?? '',
  );

  const revokeMutation = useRevokeShareLinkMutation(item?.type, item?.id);

  const handleCreate = () => {
    createMutation.mutate(
      {
        permission,
        password: password || undefined,
        expiresAt: expiresAt
          ? new Date(expiresAt + 'T23:59:59Z').toISOString()
          : undefined,
        maxAccessCount: maxAccessCount
          ? parseInt(maxAccessCount, 10)
          : undefined,
      },
      {
        onSuccess: () => {
          setPassword('');
          setExpiresAt('');
          setMaxAccessCount('');
        },
      },
    );
  };

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
          <div className="space-y-2">
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

          <div className="space-y-2">
            <Label htmlFor="share-password">Password (optional)</Label>
            <Input
              id="share-password"
              type="password"
              placeholder="Optional password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              minLength={4}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="share-expires">Expires (optional)</Label>
            <Input
              id="share-expires"
              type="date"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="share-max-access">
              Max access count (optional)
            </Label>
            <Input
              id="share-max-access"
              type="number"
              min="1"
              placeholder="Unlimited"
              value={maxAccessCount}
              onChange={(e) => setMaxAccessCount(e.target.value)}
            />
          </div>

          <Button
            className="w-full"
            onClick={handleCreate}
            disabled={createMutation.isPending}
          >
            <Link2 className="h-4 w-4 mr-2" />
            {createMutation.isPending ? 'Creating...' : 'Create Link'}
          </Button>

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
