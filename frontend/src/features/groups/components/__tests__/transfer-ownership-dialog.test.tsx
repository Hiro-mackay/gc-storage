import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { TransferOwnershipDialog } from '../transfer-ownership-dialog';

vi.mock('../../api/mutations', () => ({
  useTransferOwnershipMutation: vi.fn(),
}));

import { useTransferOwnershipMutation } from '../../api/mutations';

const mockMutate = vi.fn();
const mockUseTransferOwnershipMutation = vi.mocked(
  useTransferOwnershipMutation,
);

beforeEach(() => {
  vi.clearAllMocks();
  mockUseTransferOwnershipMutation.mockReturnValue({
    mutate: mockMutate,
    isPending: false,
  } as never);
});

function renderDialog(
  props?: Partial<Parameters<typeof TransferOwnershipDialog>[0]>,
) {
  const defaultProps = {
    groupId: 'group-1',
    targetUserId: 'user-2',
    targetUserName: 'Jane Doe',
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };
  return render(<TransferOwnershipDialog {...defaultProps} />, {
    wrapper: createWrapper(),
  });
}

describe('TransferOwnershipDialog', () => {
  it('should show confirmation message with target user name', () => {
    renderDialog();
    expect(screen.getByText(/Jane Doe/)).toBeInTheDocument();
  });

  it('should call mutation with targetUserId when confirmed', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole('button', { name: 'Transfer' }));

    expect(mockMutate).toHaveBeenCalledWith(
      'user-2',
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it('should close dialog on cancel button click', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('should show loading state during mutation', () => {
    mockUseTransferOwnershipMutation.mockReturnValue({
      mutate: mockMutate,
      isPending: true,
    } as never);

    renderDialog();

    const confirmButton = screen.getByRole('button', {
      name: 'Transferring...',
    });
    expect(confirmButton).toBeDisabled();
  });

  it('should close dialog on success', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    mockMutate.mockImplementation(
      (_vars: unknown, opts: { onSuccess?: () => void }) => {
        opts?.onSuccess?.();
      },
    );

    renderDialog({ onOpenChange });

    await user.click(screen.getByRole('button', { name: 'Transfer' }));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
