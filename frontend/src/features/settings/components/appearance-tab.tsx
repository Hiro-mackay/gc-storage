import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { useUIStore } from '@/stores/ui-store';
import { useUpdateProfileMutation } from '../api/mutations';

type Theme = 'system' | 'light' | 'dark';

export function AppearanceTab() {
  const { theme, setTheme } = useUIStore();
  const updateProfileMutation = useUpdateProfileMutation();

  const handleThemeChange = (t: Theme) => {
    setTheme(t);
    updateProfileMutation.mutate({ theme: t });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Appearance</CardTitle>
        <CardDescription>Customize the look and feel</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <Label>Theme</Label>
          <div className="flex gap-2">
            {(['system', 'light', 'dark'] as const).map((t) => (
              <Button
                key={t}
                variant={theme === t ? 'default' : 'outline'}
                size="sm"
                onClick={() => handleThemeChange(t)}
              >
                {t.charAt(0).toUpperCase() + t.slice(1)}
              </Button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
