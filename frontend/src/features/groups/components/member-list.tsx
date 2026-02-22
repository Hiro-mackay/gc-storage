import { useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { MoreHorizontal } from 'lucide-react';
import { useRemoveMemberMutation } from '../api/mutations';
import { RoleChangeDropdown } from './role-change-dropdown';
import { TransferOwnershipDialog } from './transfer-ownership-dialog';

interface Member {
  id?: string;
  userId?: string;
  name?: string;
  email?: string;
  role?: string;
  joinedAt?: string;
}

interface MemberListPanelProps {
  groupId: string;
  members: Member[];
  isOwner: boolean;
  currentUserId: string | undefined;
}

export function MemberListPanel({
  groupId,
  members,
  isOwner,
  currentUserId,
}: MemberListPanelProps) {
  const removeMemberMutation = useRemoveMemberMutation(groupId);
  const [transferTarget, setTransferTarget] = useState<{
    userId: string;
    name: string;
  } | null>(null);

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Joined</TableHead>
            {isOwner && <TableHead className="w-[50px]" />}
          </TableRow>
        </TableHeader>
        <TableBody>
          {members.map((member) => (
            <TableRow key={member.id ?? member.userId}>
              <TableCell className="font-medium">{member.name}</TableCell>
              <TableCell className="text-muted-foreground">
                {member.email}
              </TableCell>
              <TableCell>
                <Badge variant="outline">{member.role}</Badge>
              </TableCell>
              <TableCell className="text-muted-foreground text-sm">
                {member.joinedAt
                  ? new Date(member.joinedAt).toLocaleDateString()
                  : '\u2014'}
              </TableCell>
              {isOwner && (
                <TableCell>
                  {member.userId !== currentUserId && (
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <RoleChangeDropdown
                          groupId={groupId}
                          userId={member.userId ?? ''}
                          currentRole={member.role ?? ''}
                        />
                        <DropdownMenuItem
                          onClick={() =>
                            setTransferTarget({
                              userId: member.userId ?? '',
                              name: member.name ?? '',
                            })
                          }
                        >
                          Transfer Ownership
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive"
                          onClick={() => {
                            if (
                              member.userId &&
                              confirm(`Remove ${member.name ?? 'this member'}?`)
                            ) {
                              removeMemberMutation.mutate(member.userId);
                            }
                          }}
                        >
                          Remove
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </TableCell>
              )}
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {transferTarget && (
        <TransferOwnershipDialog
          groupId={groupId}
          targetUserId={transferTarget.userId}
          targetUserName={transferTarget.name}
          open={!!transferTarget}
          onOpenChange={(open) => {
            if (!open) setTransferTarget(null);
          }}
        />
      )}
    </>
  );
}
