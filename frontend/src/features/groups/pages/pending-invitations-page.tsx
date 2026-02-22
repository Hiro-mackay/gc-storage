import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
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
import { Mail } from 'lucide-react';
import { usePendingInvitations } from '../api/queries';

export function PendingInvitationsPage() {
  const { data: invitations, isLoading } = usePendingInvitations();

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-3xl space-y-6">
      <h1 className="text-2xl font-bold">Pending Invitations</h1>

      {(invitations ?? []).length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No pending invitations
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Your Invitations</CardTitle>
            <CardDescription>
              Check your email to accept or decline invitations
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Group</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Expires</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(invitations ?? []).map((item) => (
                  <TableRow key={item.invitation?.id}>
                    <TableCell className="flex items-center gap-2 font-medium">
                      <Mail className="h-4 w-4 text-muted-foreground" />
                      {item.group?.name}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{item.invitation?.role}</Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {item.invitation?.expiresAt
                        ? new Date(
                            item.invitation.expiresAt,
                          ).toLocaleDateString()
                        : '\u2014'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
