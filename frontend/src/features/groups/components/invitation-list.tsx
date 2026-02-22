import { Mail, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useGroupInvitations } from '../api/queries';
import { useCancelInvitationMutation } from '../api/mutations';

interface InvitationListPanelProps {
  groupId: string;
  canCancel: boolean;
}

export function InvitationListPanel({
  groupId,
  canCancel,
}: InvitationListPanelProps) {
  const { data: invitations } = useGroupInvitations(groupId);
  const cancelMutation = useCancelInvitationMutation(groupId);

  if (!invitations || invitations.length === 0) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pending Invitations</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Expires</TableHead>
              {canCancel && <TableHead className="w-[50px]" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {invitations.map((inv) => (
              <TableRow key={inv.id}>
                <TableCell className="flex items-center gap-2">
                  <Mail className="h-4 w-4 text-muted-foreground" />
                  {inv.email}
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{inv.role}</Badge>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {inv.expiresAt
                    ? new Date(inv.expiresAt).toLocaleDateString()
                    : '\u2014'}
                </TableCell>
                {canCancel && (
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive"
                      onClick={() => inv.id && cancelMutation.mutate(inv.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
