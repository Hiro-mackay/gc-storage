import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { api } from '@/lib/api/client';
import { groupKeys } from '@/lib/api/queries';
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
import { UserPlus, MoreHorizontal, Trash2, LogOut } from 'lucide-react';
import { toast } from 'sonner';
import { InviteMemberDialog } from '../components/invite-member-dialog';
import { InvitationListPanel } from '../components/invitation-list';

export function GroupDetailPage() {
  const params = useParams({ strict: false }) as { groupId: string };
  const groupId = params.groupId;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [inviteOpen, setInviteOpen] = useState(false);

  const { data: groupData, isLoading } = useQuery({
    queryKey: groupKeys.detail(groupId),
    queryFn: async () => {
      const { data, error } = await api.GET('/groups/{id}', {
        params: { path: { id: groupId } },
      });
      if (error) throw error;
      return data?.data;
    },
  });

  const { data: members } = useQuery({
    queryKey: groupKeys.members(groupId),
    queryFn: async () => {
      const { data, error } = await api.GET('/groups/{id}/members', {
        params: { path: { id: groupId } },
      });
      if (error) throw error;
      return data?.data ?? [];
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: async (userId: string) => {
      const { error } = await api.DELETE('/groups/{id}/members/{userId}', {
        params: { path: { id: groupId, userId } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.members(groupId) });
      toast.success('Member removed');
    },
    onError: () => {
      toast.error('Failed to remove member');
    },
  });

  const leaveMutation = useMutation({
    mutationFn: async () => {
      const { error } = await api.POST('/groups/{id}/leave', {
        params: { path: { id: groupId } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Left group');
      navigate({ to: '/groups' });
    },
    onError: () => {
      toast.error('Failed to leave group');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/groups/{id}', {
        params: { path: { id: groupId } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.all });
      toast.success('Group deleted');
      navigate({ to: '/groups' });
    },
    onError: () => {
      toast.error('Failed to delete group');
    },
  });

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
              onClick={() => leaveMutation.mutate()}
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
                  deleteMutation.mutate();
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
              {(members ?? []).map((member) => (
                <TableRow key={member.id}>
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
                      {member.userId !== user?.id && (
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                            >
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() =>
                                member.userId &&
                                removeMemberMutation.mutate(member.userId)
                              }
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
