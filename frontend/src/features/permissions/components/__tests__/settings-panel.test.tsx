import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { SettingsPanel } from '../settings-panel';

vi.mock('../../api/queries', () => ({
  useResourcePermissions: () => ({ data: [], isLoading: false }),
}));

vi.mock('../../api/mutations', () => ({
  useGrantRoleMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useRevokeGrantMutation: () => ({ mutate: vi.fn(), isPending: false }),
}));

function renderPanel(props?: Partial<Parameters<typeof SettingsPanel>[0]>) {
  const defaultProps = {
    resourceType: 'folder' as const,
    resourceId: 'folder-1',
    ...props,
  };
  return render(<SettingsPanel {...defaultProps} />, {
    wrapper: createWrapper(),
  });
}

describe('SettingsPanel', () => {
  it('should render title', () => {
    renderPanel();
    expect(screen.getByText('Sharing & Permissions')).toBeInTheDocument();
  });

  it('should display owner name when provided', () => {
    renderPanel({ ownerName: 'Alice' });
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText(/owner/i)).toBeInTheDocument();
  });

  it('should not display owner section when ownerName is omitted', () => {
    renderPanel();
    expect(screen.queryByText(/owner/i)).not.toBeInTheDocument();
  });

  it('should render Shared with label', () => {
    renderPanel();
    expect(screen.getByText(/shared with/i)).toBeInTheDocument();
  });

  it('should render Add user button', () => {
    renderPanel();
    expect(
      screen.getByRole('button', { name: /add user/i }),
    ).toBeInTheDocument();
  });

  it('should open grant dialog when Add user is clicked', async () => {
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: /add user/i }));

    expect(screen.getByText('Share with')).toBeInTheDocument();
  });

  it('should render empty state when no grants', () => {
    renderPanel();
    expect(screen.getByText(/no permissions granted yet/i)).toBeInTheDocument();
  });
});
