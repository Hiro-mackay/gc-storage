import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { NotificationsTab } from '../components/notifications-tab';

vi.mock('../api/queries', () => ({
  useProfile: vi.fn(),
}));

vi.mock('../api/mutations', () => ({
  useUpdateProfileMutation: vi.fn(),
}));

import { useProfile } from '../api/queries';
import { useUpdateProfileMutation } from '../api/mutations';

function renderTab() {
  return render(<NotificationsTab />, { wrapper: createWrapper() });
}

describe('NotificationsTab', () => {
  beforeEach(() => {
    vi.mocked(useProfile).mockReturnValue({
      data: {
        profile: {
          notification_preferences: {
            email_enabled: true,
            push_enabled: false,
          },
        },
        user: {},
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useProfile>);

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);
  });

  it('should render Email Notifications label', () => {
    renderTab();
    expect(screen.getByText('Email Notifications')).toBeInTheDocument();
  });

  it('should render Push Notifications label', () => {
    renderTab();
    expect(screen.getByText('Push Notifications')).toBeInTheDocument();
  });

  it('should render two toggle switches', () => {
    renderTab();
    const switches = screen.getAllByRole('switch');
    expect(switches).toHaveLength(2);
  });

  it('should reflect email_enabled true from profile', () => {
    renderTab();
    const switches = screen.getAllByRole('switch');
    expect(switches[0]).toHaveAttribute('data-state', 'checked');
  });

  it('should reflect push_enabled false from profile', () => {
    renderTab();
    const switches = screen.getAllByRole('switch');
    expect(switches[1]).toHaveAttribute('data-state', 'unchecked');
  });

  it('should call mutation when email toggle is clicked', async () => {
    const mutate = vi.fn();

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate,
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    renderTab();

    const switches = screen.getAllByRole('switch');
    await userEvent.click(switches[0]);

    expect(mutate).toHaveBeenCalledWith({
      notification_preferences: { email_enabled: false, push_enabled: false },
    });
  });

  it('should call mutation when push toggle is clicked', async () => {
    const mutate = vi.fn();

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate,
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    renderTab();

    const switches = screen.getAllByRole('switch');
    await userEvent.click(switches[1]);

    expect(mutate).toHaveBeenCalledWith({
      notification_preferences: { email_enabled: true, push_enabled: true },
    });
  });
});
