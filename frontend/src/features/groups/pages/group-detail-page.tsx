import { useState } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { useAuthStore } from '@/stores/auth-store';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserPlus, Trash2, LogOut } from 'lucide-react';
import { useGroupDetail, useGroupMembers } from '../api/queries';
import {
  useDeleteGroupMutation,
  useLeaveGroupMutation,
} from '../api/mutations';
import { MemberListPanel } from '../components/member-list';
import { InviteMemberDialog } from '../components/invite-member-dialog';
import { InvitationListPanel } from '../components/invitation-list';

export function GroupDetailPage() {
  const params = useParams({ strict: false }) as { groupId: string };
  const groupId = params.groupId;
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const [inviteOpen, setInviteOpen] = useState(false);

  const { data: groupData, isLoading } = useGroupDetail(groupId);
  const { data: members } = useGroupMembers(groupId);
  const deleteMutation = useDeleteGroupMutation();
  const leaveMutation = useLeaveGroupMutation();

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const group = groupData?.group;
  const myRole = groupData?.myRole;
  const isOwner = group?.ownerId === user?.id;
  const canInvite = isOwner || myRole === 'contributor';

  return (
    <div className="p-6 max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{group?.name}</h1>
          {group?.description && (
            <p className="text-muted-foreground mt-1">{group.description}</p>
          )}
        </div>
        <div className="flex gap-2">
          <Badge variant="secondary">{myRole}</Badge>
          {!isOwner && (
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                leaveMutation.mutate(groupId, {
                  onSuccess: () => navigate({ to: '/groups' }),
                })
              }
              disabled={leaveMutation.isPending}
            >
              <LogOut className="h-4 w-4 mr-1" />
              Leave
            </Button>
          )}
          {isOwner && (
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (confirm('Delete this group? This cannot be undone.')) {
                  deleteMutation.mutate(groupId, {
                    onSuccess: () => navigate({ to: '/groups' }),
                  });
                }
              }}
              disabled={deleteMutation.isPending}
            >
              <Trash2 className="h-4 w-4 mr-1" />
              Delete
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Members</CardTitle>
            <CardDescription>
              {(members ?? []).length} member
              {(members ?? []).length !== 1 ? 's' : ''}
            </CardDescription>
          </div>
          {canInvite && (
            <Button size="sm" onClick={() => setInviteOpen(true)}>
              <UserPlus className="h-4 w-4 mr-2" />
              Invite
            </Button>
          )}
        </CardHeader>
        <CardContent>
          <MemberListPanel
            groupId={groupId}
            members={members ?? []}
            isOwner={isOwner}
            currentUserId={user?.id}
          />
        </CardContent>
      </Card>

      <InvitationListPanel groupId={groupId} canCancel={isOwner} />

      <InviteMemberDialog
        groupId={groupId}
        open={inviteOpen}
        onOpenChange={setInviteOpen}
      />
    </div>
  );
}
