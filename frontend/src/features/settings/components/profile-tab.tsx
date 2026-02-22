import { useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuthStore } from '@/stores/auth-store';
import { useProfile } from '../api/queries';
import {
  useUpdateProfileMutation,
  useUpdateUserMutation,
} from '../api/mutations';
import { ChangePasswordForm } from './change-password-form';

export function ProfileTab() {
  const { user } = useAuthStore();
  const { data: profileData } = useProfile();
  const updateProfileMutation = useUpdateProfileMutation();
  const updateUserMutation = useUpdateUserMutation();

  const [displayName, setDisplayName] = useState('');
  const [bio, setBio] = useState('');
  const [locale, setLocale] = useState('');
  const [timezone, setTimezone] = useState('');
  const [prevProfileData, setPrevProfileData] = useState<typeof profileData>();

  if (profileData !== prevProfileData) {
    setPrevProfileData(profileData);
    if (profileData?.user) {
      setDisplayName(profileData.user.name ?? '');
    }
    if (profileData?.profile) {
      setBio(profileData.profile.bio ?? '');
      setLocale(profileData.profile.locale ?? '');
      setTimezone(
        profileData.profile.timezone ??
          Intl.DateTimeFormat().resolvedOptions().timeZone,
      );
    }
  }

  const isPending =
    updateProfileMutation.isPending || updateUserMutation.isPending;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    Promise.allSettled([
      updateProfileMutation.mutateAsync({ bio, locale, timezone }),
      updateUserMutation.mutateAsync({ name: displayName }),
    ]);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Manage your account details</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" value={user?.email ?? ''} disabled />
            </div>
            <div className="space-y-2">
              <Label htmlFor="displayName">Display Name</Label>
              <Input
                id="displayName"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="bio">Bio</Label>
              <Input
                id="bio"
                value={bio}
                onChange={(e) => setBio(e.target.value)}
                placeholder="A short bio about yourself"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="locale">Locale</Label>
              <Input
                id="locale"
                value={locale}
                onChange={(e) => setLocale(e.target.value)}
                placeholder="e.g. en-US"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="timezone">Timezone</Label>
              <Input
                id="timezone"
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={isPending}>
              {isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </form>
        </CardContent>
      </Card>

      <ChangePasswordForm />
    </div>
  );
}
