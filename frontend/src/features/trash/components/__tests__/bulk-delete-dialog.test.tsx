import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BulkDeleteDialog } from '../bulk-delete-dialog';

const defaultProps = {
  open: true,
  onOpenChange: vi.fn(),
  itemCount: 3,
  onConfirm: vi.fn(),
  isPending: false,
};

describe('BulkDeleteDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dialog with correct item count', () => {
    render(<BulkDeleteDialog {...defaultProps} />);

    expect(
      screen.getByRole('heading', { name: 'Permanently Delete' }),
    ).toBeInTheDocument();
    expect(screen.getByText(/3 items/)).toBeInTheDocument();
    expect(
      screen.getByText(/This action cannot be undone/),
    ).toBeInTheDocument();
  });

  it('uses singular form for single item', () => {
    render(<BulkDeleteDialog {...defaultProps} itemCount={1} />);

    expect(screen.getByText(/1 item\?/)).toBeInTheDocument();
  });

  it('does not render dialog content when closed', () => {
    render(<BulkDeleteDialog {...defaultProps} open={false} />);

    expect(screen.queryByText(/permanently delete/)).not.toBeInTheDocument();
  });

  it('calls onConfirm when delete button is clicked', async () => {
    const onConfirm = vi.fn();
    render(<BulkDeleteDialog {...defaultProps} onConfirm={onConfirm} />);

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));

    expect(onConfirm).toHaveBeenCalled();
  });

  it('calls onOpenChange when cancel is clicked', async () => {
    const onOpenChange = vi.fn();
    render(<BulkDeleteDialog {...defaultProps} onOpenChange={onOpenChange} />);

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('disables delete button when isPending', () => {
    render(<BulkDeleteDialog {...defaultProps} isPending={true} />);

    expect(screen.getByRole('button', { name: 'Deleting...' })).toBeDisabled();
  });
});
