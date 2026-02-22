import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { InviteMemberDialog } from '../invite-member-dialog';

vi.mock('../../api/mutations', () => ({
  useInviteMemberMutation: vi.fn(),
}));

import { useInviteMemberMutation } from '../../api/mutations';

const mockMutate = vi.fn();
const mockMutateAsync = vi.fn();

const mockUseMutation = vi.mocked(useInviteMemberMutation);

beforeEach(() => {
  vi.clearAllMocks();
  mockUseMutation.mockReturnValue({
    mutate: mockMutate,
    mutateAsync: mockMutateAsync,
    isPending: false,
    isError: false,
    isSuccess: false,
    error: null,
    reset: vi.fn(),
    data: undefined,
    variables: undefined,
    context: undefined,
    failureCount: 0,
    failureReason: null,
    isPaused: false,
    status: 'idle',
    submittedAt: 0,
  } as never);
});

function renderDialog(
  props?: Partial<Parameters<typeof InviteMemberDialog>[0]>,
) {
  const defaultProps = {
    groupId: 'group-1',
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };
  return render(<InviteMemberDialog {...defaultProps} />, {
    wrapper: createWrapper(),
  });
}

describe('InviteMemberDialog', () => {
  it('should render email input', () => {
    renderDialog();
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
  });

  it('should display Viewer and Contributor role options', () => {
    renderDialog();
    expect(screen.getByRole('button', { name: 'Viewer' })).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Contributor' }),
    ).toBeInTheDocument();
  });

  it('should disable submit button when email is empty', () => {
    renderDialog();
    const submitButton = screen.getByRole('button', {
      name: 'Send Invitation',
    });
    expect(submitButton).toBeDisabled();
  });

  it('should enable submit button when email is entered', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText('Email'), 'user@example.com');

    const submitButton = screen.getByRole('button', {
      name: 'Send Invitation',
    });
    expect(submitButton).not.toBeDisabled();
  });

  it('should not call mutation when email is empty and form is submitted', async () => {
    const user = userEvent.setup();
    renderDialog();

    const submitButton = screen.getByRole('button', {
      name: 'Send Invitation',
    });
    await user.click(submitButton);

    expect(mockMutate).not.toHaveBeenCalled();
  });

  it('should call mutation with email and selected role when submitted', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText('Email'), 'test@example.com');
    await user.click(screen.getByRole('button', { name: 'Send Invitation' }));

    expect(mockMutate).toHaveBeenCalledWith(
      { email: 'test@example.com', role: 'viewer' },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it('should call mutation with contributor role when contributor is selected', async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText('Email'), 'test@example.com');
    await user.click(screen.getByRole('button', { name: 'Contributor' }));
    await user.click(screen.getByRole('button', { name: 'Send Invitation' }));

    expect(mockMutate).toHaveBeenCalledWith(
      { email: 'test@example.com', role: 'contributor' },
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

  it('should show Sending... and disable submit when mutation is pending', () => {
    mockUseMutation.mockReturnValue({
      mutate: mockMutate,
      isPending: true,
    } as never);

    renderDialog();

    const submitButton = screen.getByRole('button', { name: 'Sending...' });
    expect(submitButton).toBeDisabled();
  });

  it('should close dialog and reset form on successful mutation', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    mockMutate.mockImplementation(
      (_vars: unknown, opts: { onSuccess?: () => void }) => {
        opts?.onSuccess?.();
      },
    );

    renderDialog({ onOpenChange });

    await user.type(screen.getByLabelText('Email'), 'test@example.com');
    await user.click(screen.getByRole('button', { name: 'Send Invitation' }));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
