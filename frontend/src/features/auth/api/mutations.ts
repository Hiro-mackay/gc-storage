import { useMutation } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import { useAuthStore } from '@/stores/auth-store';

export function useLoginMutation() {
  const setUser = useAuthStore((s) => s.setUser);

  return useMutation({
    mutationFn: async (input: { email: string; password: string }) => {
      const { data, error } = await api.POST('/auth/login', {
        body: input,
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Login failed');
      }
      return data!;
    },
    onSuccess: (data) => {
      if (data.data?.user) {
        setUser(data.data.user);
      }
    },
  });
}

export function useRegisterMutation() {
  const setUser = useAuthStore((s) => s.setUser);

  return useMutation({
    mutationFn: async (input: {
      email: string;
      password: string;
      name: string;
    }) => {
      const { data, error } = await api.POST('/auth/register', {
        body: input,
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Registration failed');
      }
      return data!;
    },
    onSuccess: (data) => {
      if (data.data?.user) {
        setUser(data.data.user);
      }
    },
  });
}

export function useOAuthLoginMutation() {
  const setUser = useAuthStore((s) => s.setUser);

  return useMutation({
    mutationFn: async (input: { provider: string; code: string }) => {
      const { data, error } = await api.POST('/auth/oauth/{provider}', {
        params: { path: { provider: input.provider } },
        body: { code: input.code },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Authentication failed');
      }
      return data!;
    },
    onSuccess: (data) => {
      if (data.data?.user) {
        setUser(data.data.user);
      }
    },
  });
}

export function useVerifyEmailMutation() {
  return useMutation({
    mutationFn: async (token: string) => {
      const { error } = await api.POST('/auth/email/verify', {
        params: { query: { token } },
      });
      if (error) {
        throw new Error(error.error?.message ?? 'Verification failed');
      }
    },
  });
}
