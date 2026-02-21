import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { trashKeys, folderKeys } from '@/lib/api/queries'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Trash2, RotateCcw, FileIcon, FolderIcon } from 'lucide-react'
import { toast } from 'sonner'

function getCSRFToken(): string | undefined {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/)
  return match ? decodeURIComponent(match[1]) : undefined
}

async function trashFetch(path: string, method: string) {
  const headers: Record<string, string> = {}
  const csrfToken = getCSRFToken()
  if (csrfToken) {
    headers['X-CSRF-Token'] = csrfToken
  }
  const res = await fetch(`/api/v1${path}`, {
    method,
    credentials: 'include',
    headers,
  })
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
}

function isFolder(mimeType?: string): boolean {
  return !mimeType || mimeType === 'application/x-directory'
}

export function TrashPage() {
  const queryClient = useQueryClient()
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)
  const [emptyConfirmOpen, setEmptyConfirmOpen] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: trashKeys.list(),
    queryFn: async () => {
      const { data, error } = await api.GET('/trash')
      if (error) throw error
      return data?.data?.items ?? []
    },
  })

  const restoreMutation = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.POST('/trash/{id}/restore', {
        params: { path: { id } },
        body: {},
      })
      if (error) throw error
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: trashKeys.all })
      queryClient.invalidateQueries({ queryKey: folderKeys.lists() })
      toast.success('File restored')
    },
    onError: () => {
      toast.error('Failed to restore file')
    },
  })

  const permanentDeleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await trashFetch(`/trash/${encodeURIComponent(id)}`, 'DELETE')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: trashKeys.all })
      setDeleteTarget(null)
      toast.success('Permanently deleted')
    },
    onError: () => {
      toast.error('Failed to delete')
    },
  })

  const emptyTrashMutation = useMutation({
    mutationFn: async () => {
      await trashFetch('/trash', 'DELETE')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: trashKeys.all })
      setEmptyConfirmOpen(false)
      toast.success('Trash emptied')
    },
    onError: () => {
      toast.error('Failed to empty trash')
    },
  })

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <h1 className="text-2xl font-bold">Trash</h1>
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-4">Trash</h1>
        <p className="text-destructive">Failed to load trash.</p>
      </div>
    )
  }

  const items = data ?? []

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

      {items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <Trash2 className="h-12 w-12 mb-4" />
          <p>Trash is empty</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
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
                  <div className="flex items-center gap-2">
                    {isFolder(item.mimeType) ? (
                      <FolderIcon className="h-4 w-4 text-gray-500" />
                    ) : (
                      <FileIcon className="h-4 w-4 text-gray-500" />
                    )}
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
                      onClick={() =>
                        item.id && restoreMutation.mutate(item.id)
                      }
                      disabled={restoreMutation.isPending}
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
                        setDeleteTarget({ id: item.id, name: item.name })
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
      )}

      {/* Permanent Delete Confirmation */}
      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Permanently Delete</DialogTitle>
            <DialogDescription>
              Are you sure you want to permanently delete "{deleteTarget?.name}"?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                deleteTarget && permanentDeleteMutation.mutate(deleteTarget.id)
              }
              disabled={permanentDeleteMutation.isPending}
            >
              {permanentDeleteMutation.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Empty Trash Confirmation */}
      <Dialog open={emptyConfirmOpen} onOpenChange={setEmptyConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Empty Trash</DialogTitle>
            <DialogDescription>
              Are you sure you want to permanently delete all {items.length}{' '}
              item{items.length !== 1 ? 's' : ''} in the trash? This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setEmptyConfirmOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => emptyTrashMutation.mutate()}
              disabled={emptyTrashMutation.isPending}
            >
              {emptyTrashMutation.isPending ? 'Emptying...' : 'Empty Trash'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
