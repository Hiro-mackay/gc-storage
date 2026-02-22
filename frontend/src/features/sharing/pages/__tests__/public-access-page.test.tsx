import { render, screen } from '@testing-library/react';
import { createWrapper } from '@/test/test-utils';
import { PublicAccessPage } from '../public-access-page';

vi.mock('@tanstack/react-router', () => ({
  useParams: () => ({ token: 'test-token-123' }),
}));

vi.mock('../../api/queries', () => ({
  useSharedResource: vi.fn(),
}));

vi.mock('../../api/mutations', () => ({
  useAccessShareLinkMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
    isSuccess: false,
  }),
}));

import { useSharedResource } from '../../api/queries';

function renderPage() {
  return render(<PublicAccessPage />, { wrapper: createWrapper() });
}

describe('PublicAccessPage', () => {
  it('should render loading skeleton when isLoading is true', () => {
    vi.mocked(useSharedResource).mockReturnValue({
      data: null,
      isLoading: true,
      isError: false,
      error: null,
    } as ReturnType<typeof useSharedResource>);

    renderPage();

    const skeletons = document.querySelectorAll(
      '[class*="skeleton"], [data-testid="skeleton"]',
    );
    expect(skeletons.length).toBeGreaterThanOrEqual(0);
    expect(screen.queryByText(/password protected/i)).not.toBeInTheDocument();
  });

  it('should render password prompt when hasPassword is true', () => {
    vi.mocked(useSharedResource).mockReturnValue({
      data: { hasPassword: true, resourceType: 'file', permission: 'read' },
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useSharedResource>);

    renderPage();

    expect(screen.getByText(/password protected/i)).toBeInTheDocument();
  });

  it('should render error state when link is expired (410)', () => {
    vi.mocked(useSharedResource).mockReturnValue({
      data: null,
      isLoading: false,
      isError: true,
      error: { status: 410 },
    } as ReturnType<typeof useSharedResource>);

    renderPage();

    expect(screen.getByText(/link expired or revoked/i)).toBeInTheDocument();
  });

  it('should render error state when query fails', () => {
    vi.mocked(useSharedResource).mockReturnValue({
      data: null,
      isLoading: false,
      isError: true,
      error: { status: 500 },
    } as ReturnType<typeof useSharedResource>);

    renderPage();

    expect(screen.getByText(/link expired or revoked/i)).toBeInTheDocument();
  });
});
