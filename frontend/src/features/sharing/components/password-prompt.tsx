import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Lock } from 'lucide-react';

interface PasswordPromptProps {
  onSubmit: (password: string) => void;
  isPending: boolean;
  error?: string;
}

export function PasswordPrompt({
  onSubmit,
  isPending,
  error,
}: PasswordPromptProps) {
  const [password, setPassword] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (password) {
      onSubmit(password);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-4 rounded-lg border p-6 shadow-sm">
        <div className="flex flex-col items-center gap-2 text-center">
          <Lock className="h-8 w-8 text-muted-foreground" />
          <h2 className="text-lg font-semibold">
            This link is password protected
          </h2>
          <p className="text-sm text-muted-foreground">
            Enter the password to access this shared resource.
          </p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="share-password">Password</Label>
            <Input
              id="share-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter password"
              disabled={isPending}
            />
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <Button
            type="submit"
            className="w-full"
            disabled={isPending || !password}
          >
            {isPending ? 'Verifying...' : 'Access'}
          </Button>
        </form>
      </div>
    </div>
  );
}
