import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { PasswordPrompt } from '../password-prompt';

function renderComponent(
  props: Partial<Parameters<typeof PasswordPrompt>[0]> = {},
) {
  const defaultProps = {
    onSubmit: vi.fn(),
    isPending: false,
    ...props,
  };
  return render(<PasswordPrompt {...defaultProps} />, {
    wrapper: createWrapper(),
  });
}

describe('PasswordPrompt', () => {
  it('should render password input', () => {
    renderComponent();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
  });

  it('should render Access button', () => {
    renderComponent();
    expect(screen.getByRole('button', { name: /access/i })).toBeInTheDocument();
  });

  it('should call onSubmit with password when form is submitted', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderComponent({ onSubmit });

    await user.type(screen.getByLabelText(/password/i), 'secret123');
    await user.click(screen.getByRole('button', { name: /access/i }));

    expect(onSubmit).toHaveBeenCalledWith('secret123');
  });

  it('should disable Access button when isPending is true', () => {
    renderComponent({ isPending: true });
    expect(screen.getByRole('button', { name: /verifying/i })).toBeDisabled();
  });

  it('should disable Access button when password is empty', () => {
    renderComponent();
    expect(screen.getByRole('button', { name: /access/i })).toBeDisabled();
  });

  it('should display error message when error prop is provided', () => {
    renderComponent({ error: 'Incorrect password. Please try again.' });
    expect(
      screen.getByText('Incorrect password. Please try again.'),
    ).toBeInTheDocument();
  });

  it('should render protected link message', () => {
    renderComponent();
    expect(screen.getByText(/password protected/i)).toBeInTheDocument();
  });
});
