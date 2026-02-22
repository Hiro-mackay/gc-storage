import {
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from '@/components/ui/dropdown-menu';
import { useChangeRoleMutation } from '../api/mutations';

interface RoleChangeDropdownProps {
  groupId: string;
  userId: string;
  currentRole: string;
}

export function RoleChangeDropdown({
  groupId,
  userId,
  currentRole,
}: RoleChangeDropdownProps) {
  const changeRoleMutation = useChangeRoleMutation(groupId);

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>Change Role</DropdownMenuSubTrigger>
      <DropdownMenuSubContent>
        <DropdownMenuItem
          disabled={currentRole === 'viewer' || changeRoleMutation.isPending}
          onClick={() => changeRoleMutation.mutate({ userId, role: 'viewer' })}
        >
          Viewer
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={
            currentRole === 'contributor' || changeRoleMutation.isPending
          }
          onClick={() =>
            changeRoleMutation.mutate({ userId, role: 'contributor' })
          }
        >
          Contributor
        </DropdownMenuItem>
      </DropdownMenuSubContent>
    </DropdownMenuSub>
  );
}
