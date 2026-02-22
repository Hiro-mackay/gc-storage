import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { SettingsPage } from '../pages/settings-page';

vi.mock('../api/queries', () => ({
  useProfile: vi.fn().mockReturnValue({
    data: null,
    isLoading: false,
    error: null,
  }),
}));

vi.mock('../api/mutations', () => ({
  useUpdateProfileMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useUpdateUserMutation: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({
    user: { email: 'test@example.com', name: 'Test User' },
  }),
}));

vi.mock('@/stores/ui-store', () => ({
  useUIStore: () => ({ theme: 'system', setTheme: vi.fn() }),
}));

vi.mock('../components/change-password-form', () => ({
  ChangePasswordForm: () => <div data-testid="change-password-form" />,
}));

function renderPage() {
  return render(<SettingsPage />, { wrapper: createWrapper() });
}

describe('SettingsPage', () => {
  it('should render Settings heading', () => {
    renderPage();
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('should render Profile tab trigger', () => {
    renderPage();
    expect(screen.getByRole('tab', { name: 'Profile' })).toBeInTheDocument();
  });

  it('should render Appearance tab trigger', () => {
    renderPage();
    expect(screen.getByRole('tab', { name: 'Appearance' })).toBeInTheDocument();
  });

  it('should render Notifications tab trigger', () => {
    renderPage();
    expect(
      screen.getByRole('tab', { name: 'Notifications' }),
    ).toBeInTheDocument();
  });

  it('should show Profile tab content by default', () => {
    renderPage();
    expect(screen.getByTestId('change-password-form')).toBeInTheDocument();
  });

  it('should switch to Appearance tab on click', async () => {
    renderPage();
    const appearanceTab = screen.getByRole('tab', { name: 'Appearance' });
    await userEvent.click(appearanceTab);
    expect(screen.getByText('Theme')).toBeInTheDocument();
  });

  it('should switch to Notifications tab on click', async () => {
    renderPage();
    const notificationsTab = screen.getByRole('tab', { name: 'Notifications' });
    await userEvent.click(notificationsTab);
    expect(screen.getByText('Email Notifications')).toBeInTheDocument();
  });
});
