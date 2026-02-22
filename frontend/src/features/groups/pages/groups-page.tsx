import { useState } from 'react';
import { Link } from '@tanstack/react-router';
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
import { Users, Plus, Check, X } from 'lucide-react';
import { useMyGroups, usePendingInvitations } from '../api/queries';
import {
  useAcceptInvitationMutation,
  useDeclineInvitationMutation,
} from '../api/mutations';
import { CreateGroupDialog } from '../components/create-group-dialog';

export function GroupsPage() {
  const [createOpen, setCreateOpen] = useState(false);

  const { data: groups, isLoading } = useMyGroups();
  const { data: pendingInvitations } = usePendingInvitations();
  const acceptMutation = useAcceptInvitationMutation();
  const declineMutation = useDeclineInvitationMutation();

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <h1 className="text-2xl font-bold">Groups</h1>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
    );
  }

  const invitations = pendingInvitations ?? [];

  return (
    <div className="p-6 max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Groups</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          New Group
        </Button>
      </div>

      {invitations.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold">Pending Invitations</h2>
          {invitations.map((inv) => (
            <Card key={inv.invitation?.id}>
              <CardContent className="flex items-center justify-between py-4">
                <div>
                  <p className="font-medium">{inv.group?.name}</p>
                  <p className="text-sm text-muted-foreground">
                    Role: {inv.invitation?.role}
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    onClick={() => {
                      const token = inv.invitation?.id;
                      if (token) acceptMutation.mutate(token);
                    }}
                    disabled={acceptMutation.isPending}
                  >
                    <Check className="h-4 w-4 mr-1" />
                    Accept
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      const token = inv.invitation?.id;
                      if (token) declineMutation.mutate(token);
                    }}
                    disabled={declineMutation.isPending}
                  >
                    <X className="h-4 w-4 mr-1" />
                    Decline
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {(groups ?? []).length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <Users className="h-12 w-12 mb-4" />
          <p>No groups yet</p>
          <p className="text-sm mt-1">
            Create a group to collaborate with your team
          </p>
        </div>
      ) : (
        <div className="grid gap-4">
          {(groups ?? []).map((item) => (
            <Card key={item.group?.id}>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <Link
                    to="/groups/$groupId"
                    params={{ groupId: item.group?.id ?? '' }}
                    className="hover:underline"
                  >
                    <CardTitle className="text-lg">
                      {item.group?.name}
                    </CardTitle>
                  </Link>
                  <Badge variant="secondary">{item.myRole}</Badge>
                </div>
                {item.group?.description && (
                  <CardDescription>{item.group.description}</CardDescription>
                )}
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {item.memberCount} member{item.memberCount !== 1 ? 's' : ''}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <CreateGroupDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
}
