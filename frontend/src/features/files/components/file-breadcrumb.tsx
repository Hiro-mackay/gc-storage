import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { api } from '@/lib/api/client';
import { folderKeys } from '@/lib/api/queries';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Home } from 'lucide-react';
import { Fragment } from 'react';

interface FileBreadcrumbProps {
  folderId: string | null;
  folderName?: string;
}

export function FileBreadcrumb({ folderId, folderName }: FileBreadcrumbProps) {
  const { data: ancestors } = useQuery({
    queryKey: folderKeys.ancestors(folderId ?? ''),
    queryFn: async () => {
      if (!folderId) return [];
      const { data, error } = await api.GET('/folders/{id}/ancestors', {
        params: { path: { id: folderId } },
      });
      if (error) throw error;
      return data?.data?.items ?? [];
    },
    enabled: !!folderId,
  });

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          {folderId ? (
            <BreadcrumbLink asChild>
              <Link to="/files">
                <Home className="h-4 w-4" />
              </Link>
            </BreadcrumbLink>
          ) : (
            <BreadcrumbPage className="flex items-center gap-1">
              <Home className="h-4 w-4" />
              My Files
            </BreadcrumbPage>
          )}
        </BreadcrumbItem>
        {ancestors?.map((ancestor) => (
          <Fragment key={ancestor.id}>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link
                  to="/files/$folderId"
                  params={{ folderId: ancestor.id ?? '' }}
                >
                  {ancestor.name}
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
          </Fragment>
        ))}
        {folderId && folderName && (
          <>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage>{folderName}</BreadcrumbPage>
            </BreadcrumbItem>
          </>
        )}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
