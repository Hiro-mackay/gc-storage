import { useState } from 'react';
import { useResourcePermissions } from '../api/queries';
import { useRevokeGrantMutation } from '../api/mutations';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

interface GranteeListProps {
  resourceType: 'file' | 'folder';
  resourceId: string;
}

export function GranteeList({ resourceType, resourceId }: GranteeListProps) {
  const [pendingRevokeId, setPendingRevokeId] = useState<string | null>(null);

  const { data: grants = [], isLoading } = useResourcePermissions(
    resourceType,
    resourceId,
  );
  const revokeMutation = useRevokeGrantMutation(resourceType, resourceId);

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Loading...</p>;
  }

  if (grants.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No permissions granted yet.
      </p>
    );
  }

  return (
    <ul className="space-y-2">
      {grants.map((grant) => (
        <li
          key={grant.id}
          className="flex items-center justify-between gap-2 rounded-md border p-2"
        >
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium">{grant.granteeId}</span>
            <span className="text-xs text-muted-foreground">
              {grant.granteeType}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline">{grant.role}</Badge>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                if (grant.id) {
                  setPendingRevokeId(grant.id);
                  revokeMutation.mutate(grant.id, {
                    onSettled: () => setPendingRevokeId(null),
                  });
                }
              }}
              disabled={pendingRevokeId === grant.id}
            >
              Remove
            </Button>
          </div>
        </li>
      ))}
    </ul>
  );
}
