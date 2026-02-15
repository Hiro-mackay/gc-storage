import { useQuery } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import { folderKeys } from '@/lib/api/queries'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Link } from '@tanstack/react-router'
import { Folder, FileIcon } from 'lucide-react'

export function FileBrowserPage() {
  const params = useParams({ strict: false })
  const folderId = (params as { folderId?: string }).folderId ?? null

  const { data, isLoading, error } = useQuery({
    queryKey: folderKeys.contents(folderId),
    queryFn: async () => {
      const id = folderId ?? 'root'
      const { data, error } = await api.GET('/folders/{id}/contents', {
        params: { path: { id } },
      })
      if (error) throw error
      return data?.data
    },
  })

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <p className="text-destructive">Failed to load folder contents.</p>
      </div>
    )
  }

  const folders = data?.folders ?? []
  const files = data?.files ?? []
  const isEmpty = folders.length === 0 && files.length === 0

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">
        {data?.folder?.name ?? 'My Files'}
      </h1>

      {isEmpty ? (
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <FileIcon className="h-12 w-12 mb-4" />
          <p>This folder is empty</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[50%]">Name</TableHead>
              <TableHead>Size</TableHead>
              <TableHead>Modified</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {folders.map((folder) => (
              <TableRow key={folder.id}>
                <TableCell>
                  <Link
                    to="/files/$folderId"
                    params={{ folderId: folder.id ?? '' }}
                    className="flex items-center gap-2 hover:underline"
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
            ))}
            {files.map((file) => (
              <TableRow key={file.id}>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <FileIcon className="h-4 w-4 text-gray-500" />
                    {file.name}
                  </div>
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
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}
