import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from '@/components/ui/context-menu';
import { Pencil, Trash2, Download, Share2 } from 'lucide-react';
import { toast } from 'sonner';

interface FileContextMenuProps {
  children: React.ReactNode;
  item: { id: string; name: string; type: 'file' | 'folder' };
  onRename: () => void;
  onShare: () => void;
}

export function FileContextMenu({
  children,
  item,
  onRename,
  onShare,
}: FileContextMenuProps) {
  const queryClient = useQueryClient();

  const trashMutation = useMutation({
    mutationFn: async () => {
      if (item.type === 'file') {
        const { error } = await api.POST('/files/{id}/trash', {
          params: { path: { id: item.id } },
        });
        if (error) throw error;
      } else {
        const { error } = await api.DELETE('/folders/{id}', {
          params: { path: { id: item.id } },
        });
        if (error) throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() });
      toast.success(`${item.name} moved to trash`);
    },
    onError: () => {
      toast.error('Failed to move to trash');
    },
  });

  const handleDownload = async () => {
    if (item.type !== 'file') return;
    const { data, error } = await api.GET('/files/{id}/download', {
      params: { path: { id: item.id } },
    });
    if (error) {
      toast.error('Failed to get download URL');
      return;
    }
    const url = data?.data?.url;
    if (url) {
      window.open(url, '_blank');
    }
  };

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={onRename}>
          <Pencil className="mr-2 h-4 w-4" />
          Rename
        </ContextMenuItem>
        {item.type === 'file' && (
          <ContextMenuItem onClick={handleDownload}>
            <Download className="mr-2 h-4 w-4" />
            Download
          </ContextMenuItem>
        )}
        <ContextMenuItem onClick={onShare}>
          <Share2 className="mr-2 h-4 w-4" />
          Share
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          onClick={() => trashMutation.mutate()}
          className="text-destructive"
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Move to Trash
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
