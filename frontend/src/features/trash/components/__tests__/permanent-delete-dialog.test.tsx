import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PermanentDeleteDialog } from '../permanent-delete-dialog';

const defaultProps = {
  target: { id: 'file-1', name: 'report.pdf' },
  onOpenChange: vi.fn(),
  onConfirm: vi.fn(),
  isPending: false,
};

describe('PermanentDeleteDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dialog when target is provided', () => {
    render(<PermanentDeleteDialog {...defaultProps} />);

    expect(screen.getByText('Permanently Delete')).toBeInTheDocument();
    expect(screen.getByText(/report\.pdf/)).toBeInTheDocument();
    expect(
      screen.getByText(/This action cannot be undone/),
    ).toBeInTheDocument();
  });

  it('does not render dialog content when target is null', () => {
    render(<PermanentDeleteDialog {...defaultProps} target={null} />);

    expect(screen.queryByText('Permanently Delete')).not.toBeInTheDocument();
  });

  it('calls onConfirm when delete button is clicked', async () => {
    const onConfirm = vi.fn();
    render(<PermanentDeleteDialog {...defaultProps} onConfirm={onConfirm} />);

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));

    expect(onConfirm).toHaveBeenCalled();
  });

  it('calls onOpenChange when cancel is clicked', async () => {
    const onOpenChange = vi.fn();
    render(
      <PermanentDeleteDialog {...defaultProps} onOpenChange={onOpenChange} />,
    );

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('disables delete button when isPending', () => {
    render(<PermanentDeleteDialog {...defaultProps} isPending={true} />);

    expect(screen.getByRole('button', { name: 'Deleting...' })).toBeDisabled();
  });
});
