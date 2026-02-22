import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { RoleDropdown } from './role-dropdown';
import { useGrantRoleMutation } from '../api/mutations';

type Role = 'viewer' | 'contributor' | 'content_manager';

interface GrantDialogProps {
  resourceType: 'file' | 'folder';
  resourceId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function GrantDialog({
  resourceType,
  resourceId,
  open,
  onOpenChange,
}: GrantDialogProps) {
  const [granteeType, setGranteeType] = useState<'user' | 'group'>('user');
  const [granteeId, setGranteeId] = useState('');
  const [role, setRole] = useState<Role>('viewer');

  const grantMutation = useGrantRoleMutation(resourceType, resourceId);

  const resetForm = () => {
    setGranteeType('user');
    setGranteeId('');
    setRole('viewer');
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!granteeId.trim()) return;
    grantMutation.mutate(
      { granteeType, granteeId, role },
      {
        onSuccess: () => {
          resetForm();
          onOpenChange(false);
        },
      },
    );
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share with</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="grantee-type">User/Group</Label>
              <select
                id="grantee-type"
                value={granteeType}
                onChange={(e) =>
                  setGranteeType(e.target.value as 'user' | 'group')
                }
                className="h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
              >
                <option value="user">User</option>
                <option value="group">Group</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="grantee-id">User or Group ID</Label>
              <Input
                id="grantee-id"
                value={granteeId}
                onChange={(e) => setGranteeId(e.target.value)}
                placeholder="Enter user or group ID"
              />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <RoleDropdown value={role} onChange={setRole} />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!granteeId.trim() || grantMutation.isPending}
            >
              {grantMutation.isPending ? 'Sharing...' : 'Share'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
