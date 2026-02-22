import { useState } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import {
  useAcceptInvitationMutation,
  useDeclineInvitationMutation,
} from '@/features/groups/api/mutations';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';

export function InvitationAcceptPage() {
  const params = useParams({ strict: false }) as { token: string };
  const token = params.token;
  const navigate = useNavigate();
  const [decided, setDecided] = useState(false);

  const acceptMutation = useAcceptInvitationMutation();
  const declineMutation = useDeclineInvitationMutation();

  const handleAccept = () => {
    setDecided(true);
    acceptMutation.mutate(token, {
      onSuccess: (data) => {
        const groupId = data?.data?.group?.id;
        if (groupId) {
          navigate({ to: '/groups/$groupId', params: { groupId } });
        } else {
          navigate({ to: '/groups' });
        }
      },
      onError: () => setDecided(false),
    });
  };

  const handleDecline = () => {
    setDecided(true);
    declineMutation.mutate(token, {
      onSuccess: () => {
        navigate({ to: '/invitations/pending' });
      },
      onError: () => setDecided(false),
    });
  };

  const isError = acceptMutation.isError || declineMutation.isError;
  const errorMessage =
    acceptMutation.error?.message ?? declineMutation.error?.message;

  return (
    <div className="flex items-center justify-center min-h-[60vh] p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Group Invitation</CardTitle>
          <CardDescription>
            You have been invited to join a group
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isError && (
            <div className="text-sm text-destructive mb-4">
              {errorMessage ?? 'An error occurred'}
            </div>
          )}
          {decided && !isError && (
            <div className="text-sm text-muted-foreground">Processing...</div>
          )}
        </CardContent>
        {!decided && (
          <CardFooter className="flex gap-2 justify-end">
            <Button
              variant="outline"
              onClick={handleDecline}
              disabled={declineMutation.isPending}
            >
              Decline
            </Button>
            <Button onClick={handleAccept} disabled={acceptMutation.isPending}>
              {acceptMutation.isPending ? 'Accepting...' : 'Accept'}
            </Button>
          </CardFooter>
        )}
      </Card>
    </div>
  );
}
