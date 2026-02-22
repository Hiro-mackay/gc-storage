import { useState } from 'react';
import { Link, useSearch } from '@tanstack/react-router';
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
import { PasswordStrength } from '@/components/auth/password-strength';
import { useResetPasswordMutation } from '@/features/auth/api/mutations';

export function ResetPasswordPage() {
  const search = useSearch({ strict: false }) as { token?: string };
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [mismatchError, setMismatchError] = useState('');
  const mutation = useResetPasswordMutation();

  const validationError = !search.token ? 'Reset token is missing.' : null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setMismatchError('');

    if (password !== confirmPassword) {
      setMismatchError('Passwords do not match.');
      return;
    }

    mutation.mutate({ token: search.token!, password });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Reset Password</CardTitle>
      </CardHeader>
      {validationError ? (
        <>
          <CardContent>
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {validationError}
            </div>
          </CardContent>
          <CardFooter>
            <Link to="/login" className="w-full">
              <Button className="w-full">Go to Login</Button>
            </Link>
          </CardFooter>
        </>
      ) : mutation.isSuccess ? (
        <>
          <CardContent>
            <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-700">
              Your password has been reset successfully.
            </div>
          </CardContent>
          <CardFooter>
            <Link to="/login" className="w-full">
              <Button className="w-full">Go to Login</Button>
            </Link>
          </CardFooter>
        </>
      ) : (
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            {mutation.error && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {mutation.error.message}
              </div>
            )}
            {mismatchError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {mismatchError}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="password">New Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
              <PasswordStrength password={password} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm Password</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={mutation.isPending}
            >
              {mutation.isPending ? 'Resetting...' : 'Reset Password'}
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
      )}
    </Card>
  );
}
