import { FileIcon, CheckCircle2, AlertCircle, Loader2, X } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface UploadProgressItemProps {
  item: {
    id: string;
    fileName: string;
    fileSize: number;
    progress: number;
    status: 'pending' | 'uploading' | 'completed' | 'failed';
    error?: string;
  };
  onRemove: (id: string) => void;
}

export function UploadProgressItem({
  item,
  onRemove,
}: UploadProgressItemProps) {
  const colorClass =
    item.status === 'failed'
      ? 'bg-destructive'
      : item.status === 'completed'
        ? 'bg-green-500'
        : 'bg-primary';

  return (
    <div className="flex items-center gap-3 rounded-md border p-2">
      <FileIcon className="h-4 w-4 text-muted-foreground shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate">{item.fileName}</p>
        <div className="h-1.5 bg-muted rounded-full mt-1">
          <div
            className={`h-full rounded-full transition-all ${colorClass}`}
            style={{ width: `${item.progress}%` }}
          />
        </div>
      </div>
      <div className="shrink-0">
        {item.status === 'completed' && (
          <CheckCircle2 className="h-4 w-4 text-green-500" />
        )}
        {item.status === 'failed' && (
          <AlertCircle className="h-4 w-4 text-destructive" />
        )}
        {(item.status === 'pending' || item.status === 'uploading') && (
          <Loader2 className="h-4 w-4 animate-spin text-primary" />
        )}
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6 shrink-0"
        onClick={() => onRemove(item.id)}
      >
        <X className="h-3 w-3" />
      </Button>
    </div>
  );
}
