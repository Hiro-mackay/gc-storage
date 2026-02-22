import { useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { useProfile } from '../api/queries';
import { useUpdateProfileMutation } from '../api/mutations';

export function NotificationsTab() {
  const { data: profileData } = useProfile();
  const updateProfileMutation = useUpdateProfileMutation();

  const prefs = profileData?.profile?.notification_preferences;

  const [emailEnabled, setEmailEnabled] = useState(
    prefs?.email_enabled ?? false,
  );
  const [pushEnabled, setPushEnabled] = useState(prefs?.push_enabled ?? false);
  const [prevPrefs, setPrevPrefs] = useState<typeof prefs>();

  if (prefs !== prevPrefs) {
    setPrevPrefs(prefs);
    if (prefs) {
      setEmailEnabled(prefs.email_enabled ?? false);
      setPushEnabled(prefs.push_enabled ?? false);
    }
  }

  const handleEmailToggle = (checked: boolean) => {
    setEmailEnabled(checked);
    updateProfileMutation.mutate({
      notification_preferences: {
        email_enabled: checked,
        push_enabled: pushEnabled,
      },
    });
  };

  const handlePushToggle = (checked: boolean) => {
    setPushEnabled(checked);
    updateProfileMutation.mutate({
      notification_preferences: {
        email_enabled: emailEnabled,
        push_enabled: checked,
      },
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Notifications</CardTitle>
        <CardDescription>Manage your notification preferences</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <Label htmlFor="email-notifications">Email Notifications</Label>
          <Switch
            id="email-notifications"
            checked={emailEnabled}
            onCheckedChange={handleEmailToggle}
          />
        </div>
        <div className="flex items-center justify-between">
          <Label htmlFor="push-notifications">Push Notifications</Label>
          <Switch
            id="push-notifications"
            checked={pushEnabled}
            onCheckedChange={handlePushToggle}
          />
        </div>
      </CardContent>
    </Card>
  );
}
