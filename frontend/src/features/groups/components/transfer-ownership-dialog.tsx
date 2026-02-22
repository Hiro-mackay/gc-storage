import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useTransferOwnershipMutation } from '../api/mutations';

interface TransferOwnershipDialogProps {
  groupId: string;
  targetUserId: string;
  targetUserName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function TransferOwnershipDialog({
  groupId,
  targetUserId,
  targetUserName,
  open,
  onOpenChange,
}: TransferOwnershipDialogProps) {
  const transferMutation = useTransferOwnershipMutation(groupId);

  function handleConfirm() {
    transferMutation.mutate(targetUserId, {
      onSuccess: () => onOpenChange(false),
    });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Transfer Ownership</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">
          Are you sure you want to transfer ownership to{' '}
          <span className="font-medium text-foreground">{targetUserName}</span>?
          You will become a contributor after this action.
        </p>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={transferMutation.isPending}
          >
            {transferMutation.isPending ? 'Transferring...' : 'Transfer'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
