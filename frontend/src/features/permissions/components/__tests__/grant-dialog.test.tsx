import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { GrantDialog } from '../grant-dialog';

const mockMutate = vi.fn();

vi.mock('../../api/mutations', () => ({
  useGrantRoleMutation: () => ({ mutate: mockMutate, isPending: false }),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

function renderDialog(props?: Partial<Parameters<typeof GrantDialog>[0]>) {
  const defaultProps = {
    resourceType: 'file' as const,
    resourceId: 'file-1',
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };
  return render(<GrantDialog {...defaultProps} />, {
    wrapper: createWrapper(),
  });
}

describe('GrantDialog', () => {
  it('should render dialog with title when open', () => {
    renderDialog();
    expect(screen.getByText('Share with')).toBeInTheDocument();
  });

  it('should render grantee type selector', () => {
    renderDialog();
    expect(screen.getByLabelText(/user\/group/i)).toBeInTheDocument();
  });

  it('should render grantee id input', () => {
    renderDialog();
    expect(screen.getByLabelText(/user or group id/i)).toBeInTheDocument();
  });

  it('should render role dropdown', () => {
    renderDialog();
    const comboboxes = screen.getAllByRole('combobox');
    expect(comboboxes.length).toBeGreaterThanOrEqual(1);
  });

  it('should render Cancel and Share buttons', () => {
    renderDialog();
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /share/i })).toBeInTheDocument();
  });

  it('should disable Share button when granteeId is empty', () => {
    renderDialog();
    expect(screen.getByRole('button', { name: /share/i })).toBeDisabled();
  });

  it('should enable Share button when granteeId is filled', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText(/user or group id/i), 'some-uuid');

    expect(screen.getByRole('button', { name: /share/i })).not.toBeDisabled();
  });

  it('should call mutation with correct params on submit', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText(/user or group id/i), 'user-uuid-1');
    await user.click(screen.getByRole('button', { name: /share/i }));

    expect(mockMutate).toHaveBeenCalledWith(
      {
        granteeType: 'user',
        granteeId: 'user-uuid-1',
        role: 'viewer',
      },
      expect.any(Object),
    );
  });

  it('should call onOpenChange(false) when Cancel is clicked', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    await user.click(screen.getByRole('button', { name: /cancel/i }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('should not render when open is false', () => {
    renderDialog({ open: false });
    expect(screen.queryByText('Share with')).not.toBeInTheDocument();
  });
});
