import { useEffect, useRef, useState } from 'react';
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
import {
  useVerifyEmailMutation,
  useResendVerificationMutation,
} from '@/features/auth/api/mutations';

export function VerifyEmailPage() {
  const search = useSearch({ strict: false }) as { token?: string };
  const mutation = useVerifyEmailMutation();
  const resendMutation = useResendVerificationMutation();
  const processedRef = useRef(false);
  const [resendEmail, setResendEmail] = useState('');

  // Derive validation error synchronously (not in an effect)
  const validationError = !search.token
    ? 'Verification token is missing.'
    : null;

  useEffect(() => {
    if (validationError || !search.token || processedRef.current) return;
    processedRef.current = true;

    mutation.mutate(search.token);
    // mutation.mutate is a stable reference
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validationError, search.token]);

  const isPending = !validationError && mutation.isPending;
  const error =
    validationError ?? (mutation.error ? mutation.error.message : null);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Email Verification</CardTitle>
      </CardHeader>
      <CardContent>
        {isPending && (
          <p className="text-sm text-muted-foreground">
            Verifying your email...
          </p>
        )}
        {mutation.isSuccess && (
          <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-700">
            Your email has been verified successfully. You can now log in.
          </div>
        )}
        {error && (
          <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            {error}
          </div>
        )}
        {mutation.isError && (
          <div className="mt-4 space-y-3">
            <p className="text-sm text-muted-foreground">
              Didn&apos;t receive the email? Resend verification.
            </p>
            <div className="space-y-2">
              <Label htmlFor="resend-email">Email</Label>
              <Input
                id="resend-email"
                type="email"
                placeholder="you@example.com"
                value={resendEmail}
                onChange={(e) => setResendEmail(e.target.value)}
              />
            </div>
            <Button
              className="w-full"
              variant="outline"
              disabled={!resendEmail || resendMutation.isPending}
              onClick={() => resendMutation.mutate({ email: resendEmail })}
            >
              {resendMutation.isPending
                ? 'Sending...'
                : 'Resend Verification Email'}
            </Button>
            {resendMutation.isSuccess && (
              <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-700">
                Verification email sent. Please check your inbox.
              </div>
            )}
            {resendMutation.error && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {resendMutation.error.message}
              </div>
            )}
          </div>
        )}
      </CardContent>
      <CardFooter>
        <Link to="/login" className="w-full">
          <Button className="w-full">Go to Login</Button>
        </Link>
      </CardFooter>
    </Card>
  );
}
