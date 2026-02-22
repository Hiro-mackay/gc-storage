import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { CreateGroupDialog } from '../create-group-dialog';

vi.mock('../../api/mutations', () => ({
  useCreateGroupMutation: vi.fn(),
}));

import { useCreateGroupMutation } from '../../api/mutations';

const mockMutate = vi.fn();
const mockUseCreateGroupMutation = vi.mocked(useCreateGroupMutation);

beforeEach(() => {
  vi.clearAllMocks();
  mockUseCreateGroupMutation.mockReturnValue({
    mutate: mockMutate,
    isPending: false,
  } as never);
});

function renderDialog(
  props?: Partial<Parameters<typeof CreateGroupDialog>[0]>,
) {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };
  return render(<CreateGroupDialog {...defaultProps} />, {
    wrapper: createWrapper(),
  });
}

describe('CreateGroupDialog', () => {
  it('should render name and description inputs', () => {
    renderDialog();
    expect(screen.getByLabelText('Name')).toBeInTheDocument();
    expect(screen.getByLabelText('Description (optional)')).toBeInTheDocument();
  });

  it('should disable submit button when name is empty', () => {
    renderDialog();
    const submitButton = screen.getByRole('button', { name: 'Create' });
    expect(submitButton).toBeDisabled();
  });

  it('should enable submit button when name is entered', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText('Name'), 'My Team');

    expect(screen.getByRole('button', { name: 'Create' })).not.toBeDisabled();
  });

  it('should call mutation with name and description on submit', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText('Name'), 'My Team');
    await user.type(
      screen.getByLabelText('Description (optional)'),
      'A cool team',
    );
    await user.click(screen.getByRole('button', { name: 'Create' }));

    expect(mockMutate).toHaveBeenCalledWith(
      { name: 'My Team', description: 'A cool team' },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it('should call mutation with name only when description is empty', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText('Name'), 'My Team');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    expect(mockMutate).toHaveBeenCalledWith(
      { name: 'My Team', description: undefined },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it('should close dialog and reset form on success', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    mockMutate.mockImplementation(
      (_vars: unknown, opts: { onSuccess?: () => void }) => {
        opts?.onSuccess?.();
      },
    );

    renderDialog({ onOpenChange });

    await user.type(screen.getByLabelText('Name'), 'My Team');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it('should close dialog on cancel button click', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('should show Creating... and disable submit when mutation is pending', () => {
    mockUseCreateGroupMutation.mockReturnValue({
      mutate: mockMutate,
      isPending: true,
    } as never);

    renderDialog();

    const submitButton = screen.getByRole('button', { name: 'Creating...' });
    expect(submitButton).toBeDisabled();
  });
});
