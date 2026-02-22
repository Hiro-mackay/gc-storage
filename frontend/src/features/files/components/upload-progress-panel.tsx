import { useUploadStore } from '@/stores/upload-store';
import { Button } from '@/components/ui/button';
import { UploadProgressItem } from './upload-progress-item';

export function UploadProgressPanel() {
  const uploads = useUploadStore((s) => s.uploads);
  const removeUpload = useUploadStore((s) => s.removeUpload);
  const clearCompleted = useUploadStore((s) => s.clearCompleted);

  const uploadItems = Array.from(uploads.values());
  const activeNum = uploadItems.filter(
    (i) => i.status === 'pending' || i.status === 'uploading',
  ).length;

  if (uploadItems.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 w-80 rounded-lg border bg-background shadow-lg">
      <div className="flex items-center justify-between p-3 border-b">
        <span className="text-sm font-medium">
          Uploading {activeNum} file(s)
        </span>
        <Button variant="ghost" size="sm" onClick={clearCompleted}>
          Clear
        </Button>
      </div>
      <div className="max-h-60 overflow-auto p-2 space-y-2">
        {uploadItems.map((item) => (
          <UploadProgressItem
            key={item.id}
            item={item}
            onRemove={removeUpload}
          />
        ))}
      </div>
    </div>
  );
}
