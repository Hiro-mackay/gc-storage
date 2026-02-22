import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { GranteeList } from './grantee-list';
import { GrantDialog } from './grant-dialog';

interface SettingsPanelProps {
  resourceType: 'file' | 'folder';
  resourceId: string;
  ownerName?: string;
}

export function SettingsPanel({
  resourceType,
  resourceId,
  ownerName,
}: SettingsPanelProps) {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <div className="space-y-4 rounded-lg border p-4">
      <h3 className="font-semibold">Sharing &amp; Permissions</h3>

      {ownerName && (
        <div className="text-sm text-muted-foreground">
          Owner:{' '}
          <span className="font-medium text-foreground">{ownerName}</span>
        </div>
      )}

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">Shared with:</span>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setDialogOpen(true)}
          >
            + Add user
          </Button>
        </div>

        <GranteeList resourceType={resourceType} resourceId={resourceId} />
      </div>

      <GrantDialog
        resourceType={resourceType}
        resourceId={resourceId}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </div>
  );
}
