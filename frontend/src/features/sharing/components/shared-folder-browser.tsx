import { File, Folder, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface FolderItem {
  id: string;
  name: string;
  type: 'file' | 'folder';
  size?: number;
  mimeType?: string;
}

interface SharedFolderBrowserProps {
  contents: FolderItem[];
  permission: string;
  token: string;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function SharedFolderBrowser({
  contents,
  token,
}: SharedFolderBrowserProps) {
  const getDownloadUrl = (fileId: string) =>
    `/api/v1/share/${token}/download?fileId=${fileId}`;

  if (contents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <Folder className="h-12 w-12 mb-2" />
        <p className="text-sm">This folder is empty</p>
      </div>
    );
  }

  return (
    <div className="divide-y rounded-md border">
      {contents.map((item) => (
        <div
          key={item.id}
          className="flex items-center gap-3 px-4 py-3 hover:bg-muted/50"
        >
          {item.type === 'folder' ? (
            <Folder className="h-5 w-5 shrink-0 text-muted-foreground" />
          ) : (
            <File className="h-5 w-5 shrink-0 text-muted-foreground" />
          )}
          <div className="flex-1 min-w-0">
            <p className="truncate text-sm font-medium">{item.name}</p>
            {item.type === 'file' && item.size !== undefined && (
              <p className="text-xs text-muted-foreground">
                {formatBytes(item.size)}
              </p>
            )}
          </div>
          {item.type === 'file' && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 shrink-0"
              asChild
            >
              <a
                href={getDownloadUrl(item.id)}
                download={item.name}
                aria-label={`Download ${item.name}`}
              >
                <Download className="h-4 w-4" />
              </a>
            </Button>
          )}
        </div>
      ))}
    </div>
  );
}
