import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useUIStore } from '@/stores/ui-store';
import {
  Upload,
  FolderPlus,
  LayoutGrid,
  LayoutList,
  ArrowUpDown,
} from 'lucide-react';

interface FileToolbarProps {
  onUpload: () => void;
  onCreateFolder: () => void;
}

export function FileToolbar({ onUpload, onCreateFolder }: FileToolbarProps) {
  const { viewMode, setViewMode, sortBy, setSortBy, sortOrder, setSortOrder } =
    useUIStore();

  const sortOptions = [
    { value: 'name' as const, label: 'Name' },
    { value: 'updatedAt' as const, label: 'Modified' },
    { value: 'size' as const, label: 'Size' },
  ];

  return (
    <div className="flex items-center gap-2">
      <Button onClick={onUpload} size="sm">
        <Upload className="h-4 w-4 mr-2" />
        Upload
      </Button>
      <Button onClick={onCreateFolder} variant="outline" size="sm">
        <FolderPlus className="h-4 w-4 mr-2" />
        New Folder
      </Button>
      <div className="flex-1" />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm">
            <ArrowUpDown className="h-4 w-4 mr-1" />
            Sort
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {sortOptions.map((opt) => (
            <DropdownMenuItem
              key={opt.value}
              onClick={() => {
                if (sortBy === opt.value) {
                  setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
                } else {
                  setSortBy(opt.value);
                  setSortOrder('asc');
                }
              }}
            >
              {opt.label}
              {sortBy === opt.value && (sortOrder === 'asc' ? ' ↑' : ' ↓')}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setViewMode(viewMode === 'list' ? 'grid' : 'list')}
      >
        {viewMode === 'list' ? (
          <LayoutGrid className="h-4 w-4" />
        ) : (
          <LayoutList className="h-4 w-4" />
        )}
      </Button>
    </div>
  );
}
