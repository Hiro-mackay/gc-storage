import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TrashList } from '../trash-list';
import type { TrashItem } from '../../api/queries';

const mockItems: TrashItem[] = [
  {
    id: 'file-1',
    name: 'report.pdf',
    mimeType: 'application/pdf',
    originalPath: '/documents',
    archivedAt: '2026-01-25T00:00:00Z',
    daysUntilExpiry: 5,
  },
  {
    id: 'file-2',
    name: 'data.csv',
    mimeType: 'text/csv',
    originalPath: '/exports',
    archivedAt: '2026-01-20T00:00:00Z',
    daysUntilExpiry: 2,
  },
];

const defaultProps = {
  items: mockItems,
  selectedIds: new Set<string>(),
  onSelectionChange: vi.fn(),
  onRestore: vi.fn(),
  onDelete: vi.fn(),
  isRestoring: false,
};

describe('TrashList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders all items in the table', () => {
    render(<TrashList {...defaultProps} />);

    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    expect(screen.getByText('data.csv')).toBeInTheDocument();
    expect(screen.getByText('/documents')).toBeInTheDocument();
    expect(screen.getByText('/exports')).toBeInTheDocument();
  });

  it('renders expiry badges with correct variant', () => {
    render(<TrashList {...defaultProps} />);

    expect(screen.getByText('5d left')).toBeInTheDocument();
    expect(screen.getByText('2d left')).toBeInTheDocument();
  });

  it('calls onSelectionChange when a checkbox is clicked', async () => {
    const onSelectionChange = vi.fn();
    render(
      <TrashList {...defaultProps} onSelectionChange={onSelectionChange} />,
    );

    const checkboxes = screen.getAllByRole('checkbox');
    await userEvent.click(checkboxes[1]); // first item checkbox

    expect(onSelectionChange).toHaveBeenCalledWith(new Set(['file-1']));
  });

  it('calls onSelectionChange with all ids when select all is clicked', async () => {
    const onSelectionChange = vi.fn();
    render(
      <TrashList {...defaultProps} onSelectionChange={onSelectionChange} />,
    );

    const checkboxes = screen.getAllByRole('checkbox');
    await userEvent.click(checkboxes[0]); // select all checkbox

    expect(onSelectionChange).toHaveBeenCalledWith(
      new Set(['file-1', 'file-2']),
    );
  });

  it('calls onRestore when restore button is clicked', async () => {
    const onRestore = vi.fn();
    render(<TrashList {...defaultProps} onRestore={onRestore} />);

    const restoreButtons = screen.getAllByTitle('Restore');
    await userEvent.click(restoreButtons[0]);

    expect(onRestore).toHaveBeenCalledWith('file-1');
  });

  it('calls onDelete when delete button is clicked', async () => {
    const onDelete = vi.fn();
    render(<TrashList {...defaultProps} onDelete={onDelete} />);

    const deleteButtons = screen.getAllByTitle('Permanently Delete');
    await userEvent.click(deleteButtons[0]);

    expect(onDelete).toHaveBeenCalledWith({ id: 'file-1', name: 'report.pdf' });
  });

  it('disables restore buttons when isRestoring is true', () => {
    render(<TrashList {...defaultProps} isRestoring={true} />);

    const restoreButtons = screen.getAllByTitle('Restore');
    restoreButtons.forEach((btn) => {
      expect(btn).toBeDisabled();
    });
  });
});
