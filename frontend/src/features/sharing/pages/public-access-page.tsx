import { useState, useEffect, useRef } from 'react';
import { useParams } from '@tanstack/react-router';
import { AlertCircle, Download, File } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { PasswordPrompt } from '../components/password-prompt';
import { SharedFolderBrowser } from '../components/shared-folder-browser';
import { useSharedResource } from '../api/queries';
import { useAccessShareLinkMutation } from '../api/mutations';

interface FolderItem {
  id: string;
  name: string;
  type: 'file' | 'folder';
  size?: number;
  mimeType?: string;
}

interface AccessResult {
  resourceType?: string;
  resourceId?: string;
  permission?: string;
  resourceName?: string;
  contents?: FolderItem[];
}

export function PublicAccessPage() {
  const { token } = useParams({ strict: false }) as { token: string };
  const [accessResult, setAccessResult] = useState<AccessResult | null>(null);
  const [accessError, setAccessError] = useState<string | undefined>(undefined);

  const sharedResourceQuery = useSharedResource(token);
  const accessMutation = useAccessShareLinkMutation(token);
  const autoAccessTriggered = useRef(false);

  const linkInfo = sharedResourceQuery.data;
  const isExpired =
    sharedResourceQuery.error &&
    (sharedResourceQuery.error as { status?: number }).status === 410;

  const { mutate: doAccess } = accessMutation;

  useEffect(() => {
    if (linkInfo && !linkInfo.hasPassword && !autoAccessTriggered.current) {
      autoAccessTriggered.current = true;
      doAccess(undefined, {
        onSuccess: (result) => {
          setAccessResult((result as AccessResult) ?? null);
        },
      });
    }
  }, [linkInfo, doAccess]);

  const handlePasswordSubmit = (password: string) => {
    setAccessError(undefined);
    accessMutation.mutate(password, {
      onSuccess: (result) => {
        setAccessResult((result as AccessResult) ?? null);
      },
      onError: () => {
        setAccessError('Incorrect password. Please try again.');
      },
    });
  };

  if (sharedResourceQuery.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="w-full max-w-lg space-y-4 p-6">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-10 w-32" />
        </div>
      </div>
    );
  }

  if (isExpired || sharedResourceQuery.isError) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3 text-center">
          <AlertCircle className="h-12 w-12 text-destructive" />
          <h2 className="text-xl font-semibold">Link expired or revoked</h2>
          <p className="text-sm text-muted-foreground">
            This link has expired or been revoked.
          </p>
        </div>
      </div>
    );
  }

  if (linkInfo?.hasPassword && !accessResult) {
    return (
      <PasswordPrompt
        onSubmit={handlePasswordSubmit}
        isPending={accessMutation.isPending}
        error={accessError}
      />
    );
  }

  if (!accessResult) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="w-full max-w-lg space-y-4 p-6">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-full" />
        </div>
      </div>
    );
  }

  const isFile = accessResult.resourceType === 'file';
  const downloadUrl = `/api/v1/share/${token}/download`;

  if (isFile) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="w-full max-w-sm space-y-4 rounded-lg border p-6 shadow-sm">
          <div className="flex flex-col items-center gap-3 text-center">
            <File className="h-12 w-12 text-muted-foreground" />
            <h2 className="text-lg font-semibold">
              {accessResult.resourceName ?? 'Shared File'}
            </h2>
            <p className="text-sm text-muted-foreground">
              Permission: {accessResult.permission}
            </p>
          </div>
          <Button className="w-full" asChild>
            <a href={downloadUrl} download>
              <Download className="mr-2 h-4 w-4" />
              Download
            </a>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-2xl px-4 py-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold">
            {accessResult.resourceName ?? 'Shared Folder'}
          </h1>
          <p className="text-sm text-muted-foreground">
            Permission: {accessResult.permission}
          </p>
        </div>
        <SharedFolderBrowser
          contents={accessResult.contents ?? []}
          permission={accessResult.permission ?? 'read'}
          token={token}
        />
      </div>
    </div>
  );
}
