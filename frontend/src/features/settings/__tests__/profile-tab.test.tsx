import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { ProfileTab } from '../components/profile-tab';

vi.mock('../api/queries', () => ({
  useProfile: vi.fn(),
}));

vi.mock('../api/mutations', () => ({
  useUpdateProfileMutation: vi.fn(),
  useUpdateUserMutation: vi.fn(),
}));

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({
    user: { email: 'test@example.com', name: 'Test User' },
  }),
}));

vi.mock('../components/change-password-form', () => ({
  ChangePasswordForm: () => <div data-testid="change-password-form" />,
}));

import { useProfile } from '../api/queries';
import {
  useUpdateProfileMutation,
  useUpdateUserMutation,
} from '../api/mutations';

function renderTab() {
  return render(<ProfileTab />, { wrapper: createWrapper() });
}

const mockProfile = {
  profile: {
    bio: 'A bio',
    locale: 'en',
    timezone: 'UTC',
  },
  user: {
    email: 'test@example.com',
    name: 'Test User',
  },
};

describe('ProfileTab', () => {
  beforeEach(() => {
    vi.mocked(useProfile).mockReturnValue({
      data: mockProfile,
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useProfile>);

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn().mockResolvedValue(undefined),
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    vi.mocked(useUpdateUserMutation).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn().mockResolvedValue(undefined),
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateUserMutation>);
  });

  it('should render email field as read-only', () => {
    renderTab();
    const emailInput = screen.getByLabelText('Email');
    expect(emailInput).toBeDisabled();
    expect(emailInput).toHaveValue('test@example.com');
  });

  it('should render Display Name field', () => {
    renderTab();
    expect(screen.getByLabelText('Display Name')).toBeInTheDocument();
  });

  it('should render Bio field', () => {
    renderTab();
    expect(screen.getByLabelText('Bio')).toBeInTheDocument();
  });

  it('should render Timezone field', () => {
    renderTab();
    expect(screen.getByLabelText('Timezone')).toBeInTheDocument();
  });

  it('should render Save Changes button', () => {
    renderTab();
    expect(
      screen.getByRole('button', { name: 'Save Changes' }),
    ).toBeInTheDocument();
  });

  it('should show Saving... when pending', () => {
    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn().mockResolvedValue(undefined),
      isPending: true,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    renderTab();
    expect(
      screen.getByRole('button', { name: 'Saving...' }),
    ).toBeInTheDocument();
  });

  it('should populate fields from profile data', () => {
    renderTab();
    expect(screen.getByLabelText('Bio')).toHaveValue('A bio');
    expect(screen.getByLabelText('Timezone')).toHaveValue('UTC');
    expect(screen.getByLabelText('Display Name')).toHaveValue('Test User');
  });

  it('should call mutations on Save Changes click', async () => {
    const profileMutateAsync = vi.fn().mockResolvedValue(undefined);
    const userMutateAsync = vi.fn().mockResolvedValue(undefined);

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: profileMutateAsync,
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    vi.mocked(useUpdateUserMutation).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: userMutateAsync,
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateUserMutation>);

    renderTab();

    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    expect(profileMutateAsync).toHaveBeenCalled();
    expect(userMutateAsync).toHaveBeenCalled();
  });

  it('should render ChangePasswordForm', () => {
    renderTab();
    expect(screen.getByTestId('change-password-form')).toBeInTheDocument();
  });
});
