import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EmptyTrashDialog } from '../empty-trash-dialog';

const defaultProps = {
  open: true,
  onOpenChange: vi.fn(),
  itemCount: 5,
  onConfirm: vi.fn(),
  isPending: false,
};

describe('EmptyTrashDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dialog with correct item count', () => {
    render(<EmptyTrashDialog {...defaultProps} />);

    expect(
      screen.getByRole('heading', { name: 'Empty Trash' }),
    ).toBeInTheDocument();
    expect(screen.getByText(/5 items/)).toBeInTheDocument();
    expect(
      screen.getByText(/This action cannot be undone/),
    ).toBeInTheDocument();
  });

  it('uses singular form for single item', () => {
    render(<EmptyTrashDialog {...defaultProps} itemCount={1} />);

    expect(screen.getByText(/1 item in/)).toBeInTheDocument();
  });

  it('does not render dialog content when closed', () => {
    render(<EmptyTrashDialog {...defaultProps} open={false} />);

    expect(
      screen.queryByText(/permanently delete all/),
    ).not.toBeInTheDocument();
  });

  it('calls onConfirm when Empty Trash button is clicked', async () => {
    const onConfirm = vi.fn();
    render(<EmptyTrashDialog {...defaultProps} onConfirm={onConfirm} />);

    await userEvent.click(screen.getByRole('button', { name: 'Empty Trash' }));

    expect(onConfirm).toHaveBeenCalled();
  });

  it('calls onOpenChange when cancel is clicked', async () => {
    const onOpenChange = vi.fn();
    render(<EmptyTrashDialog {...defaultProps} onOpenChange={onOpenChange} />);

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('disables Empty Trash button when isPending', () => {
    render(<EmptyTrashDialog {...defaultProps} isPending={true} />);

    expect(screen.getByRole('button', { name: 'Emptying...' })).toBeDisabled();
  });
});
