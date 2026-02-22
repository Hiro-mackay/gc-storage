import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useForgotPasswordMutation } from '@/features/auth/api/mutations';

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const mutation = useForgotPasswordMutation();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate({ email });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Forgot Password</CardTitle>
      </CardHeader>
      <form onSubmit={handleSubmit}>
        <CardContent className="space-y-4">
          {mutation.error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {mutation.error.message}
            </div>
          )}
          {mutation.isSuccess && (
            <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-700">
              If an account exists for {email}, we&apos;ve sent a password reset
              link.
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={mutation.isSuccess}
              required
            />
          </div>
          <Button
            type="submit"
            className="w-full"
            disabled={mutation.isPending || mutation.isSuccess}
          >
            {mutation.isPending ? 'Sending...' : 'Send Reset Link'}
          </Button>
        </CardContent>
        <CardFooter>
          <p className="w-full text-center text-sm text-muted-foreground">
            <Link to="/login" className="text-primary hover:underline">
              Back to Login
            </Link>
          </p>
        </CardFooter>
      </form>
    </Card>
  );
}
