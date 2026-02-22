import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { profileKeys } from '@/lib/api/queries';

// Response shape from GET /me/profile after backend restructuring.
// Will be auto-typed once schema is regenerated via `task api:generate`.
export interface ProfileResponse {
  profile: {
    id: string;
    user_id: string;
    avatar_url?: string;
    bio?: string;
    locale: string;
    timezone: string;
    theme: string;
    notification_preferences?: {
      email_enabled?: boolean;
      push_enabled?: boolean;
    };
    updated_at: string;
  };
  user: {
    id: string;
    email: string;
    name: string;
    status: string;
    email_verified: boolean;
    created_at: string;
    updated_at: string;
  };
}

export function useProfile() {
  return useQuery({
    queryKey: profileKeys.all,
    queryFn: async () => {
      const { data, error } = await api.GET('/me/profile');
      if (error) throw error;
      // Cast needed until schema is regenerated via `task api:generate`
      return (data?.data as unknown as ProfileResponse) ?? null;
    },
  });
}
