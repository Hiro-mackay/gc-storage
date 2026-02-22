import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { TrashPage } from '../pages/trash-page';

vi.mock('../api/queries', () => ({
  useTrashItems: vi.fn(),
}));

vi.mock('../api/mutations', () => ({
  useRestoreFileMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  }),
  usePermanentDeleteMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  }),
  useEmptyTrashMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

import { useTrashItems } from '../api/queries';

function renderPage() {
  return render(<TrashPage />, { wrapper: createWrapper() });
}

const mockItems = [
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

describe('TrashPage', () => {
  it('should display "Trash is empty" when there are no items', () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: { items: [] },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    expect(screen.getByText('Trash is empty')).toBeInTheDocument();
  });

  it('should render items in table', () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: { items: mockItems },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    expect(screen.getByText('data.csv')).toBeInTheDocument();
    expect(screen.getByText('/documents')).toBeInTheDocument();
    expect(screen.getByText('/exports')).toBeInTheDocument();
  });

  it('should toggle selection checkbox', async () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: { items: mockItems },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    const checkboxes = screen.getAllByRole('checkbox');
    // First checkbox is "Select all", rest are per-item
    expect(checkboxes).toHaveLength(3);

    const firstItemCheckbox = checkboxes[1];
    await userEvent.click(firstItemCheckbox);

    // After clicking, the toolbar should appear with "1 item selected"
    expect(screen.getByText('1 item selected')).toBeInTheDocument();
  });

  it('should show loading skeletons when loading', () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    expect(screen.getByText('Trash')).toBeInTheDocument();
    expect(screen.queryByText('Trash is empty')).not.toBeInTheDocument();
  });

  it('should show error message on failure', () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Network error'),
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    expect(screen.getByText('Failed to load trash.')).toBeInTheDocument();
  });

  it('should show 30-day notice', () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: { items: mockItems },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    expect(
      screen.getByText(
        'Items in trash will be automatically deleted after 30 days.',
      ),
    ).toBeInTheDocument();
  });

  it('should show Empty Trash button when items exist', () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: { items: mockItems },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    expect(screen.getByText('Empty Trash')).toBeInTheDocument();
  });

  it('should show bulk delete confirmation when Delete permanently is clicked from toolbar', async () => {
    vi.mocked(useTrashItems).mockReturnValue({
      data: { items: mockItems },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useTrashItems>);

    renderPage();

    // Select an item first
    const checkboxes = screen.getAllByRole('checkbox');
    await userEvent.click(checkboxes[1]);

    // Click "Delete permanently" in the toolbar
    await userEvent.click(screen.getByText('Delete permanently'));

    // Bulk delete confirmation dialog should appear
    expect(
      screen.getByText(/Are you sure you want to permanently delete 1 item\?/),
    ).toBeInTheDocument();
  });
});
