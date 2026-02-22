import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { RoleChangeDropdown } from '../role-change-dropdown';

vi.mock('../../api/mutations', () => ({
  useChangeRoleMutation: vi.fn(),
}));

import { useChangeRoleMutation } from '../../api/mutations';

const mockMutate = vi.fn();
const mockUseChangeRoleMutation = vi.mocked(useChangeRoleMutation);

beforeEach(() => {
  vi.clearAllMocks();
  mockUseChangeRoleMutation.mockReturnValue({
    mutate: mockMutate,
    isPending: false,
  } as never);
});

function renderDropdown(
  props?: Partial<Parameters<typeof RoleChangeDropdown>[0]>,
) {
  const defaultProps = {
    groupId: 'group-1',
    userId: 'user-2',
    currentRole: 'viewer',
    ...props,
  };
  return render(
    <DropdownMenu defaultOpen>
      <DropdownMenuTrigger>Open</DropdownMenuTrigger>
      <DropdownMenuContent>
        <RoleChangeDropdown {...defaultProps} />
      </DropdownMenuContent>
    </DropdownMenu>,
    { wrapper: createWrapper() },
  );
}

async function openSubMenu() {
  const subTrigger = screen.getByText('Change Role');
  subTrigger.focus();
  fireEvent.keyDown(subTrigger, { key: 'ArrowRight' });
}

describe('RoleChangeDropdown', () => {
  it('should render Change Role sub-trigger', () => {
    renderDropdown();
    expect(screen.getByText('Change Role')).toBeInTheDocument();
  });

  it('should show viewer and contributor options in sub-menu', async () => {
    renderDropdown();
    await openSubMenu();
    expect(screen.getByText('Viewer')).toBeInTheDocument();
    expect(screen.getByText('Contributor')).toBeInTheDocument();
  });

  it('should not show owner as a role option', async () => {
    renderDropdown();
    await openSubMenu();
    expect(screen.queryByText('Owner')).not.toBeInTheDocument();
  });

  it('should disable viewer option when currentRole is viewer', async () => {
    renderDropdown({ currentRole: 'viewer' });
    await openSubMenu();
    const viewerItem = screen.getByText('Viewer').closest('[role="menuitem"]');
    expect(viewerItem).toHaveAttribute('aria-disabled', 'true');
  });

  it('should disable contributor option when currentRole is contributor', async () => {
    renderDropdown({ currentRole: 'contributor' });
    await openSubMenu();
    const contributorItem = screen
      .getByText('Contributor')
      .closest('[role="menuitem"]');
    expect(contributorItem).toHaveAttribute('aria-disabled', 'true');
  });

  it('should call mutation with viewer role when viewer is clicked', async () => {
    const user = userEvent.setup();
    renderDropdown({ currentRole: 'contributor' });
    await openSubMenu();

    await user.click(screen.getByText('Viewer'));

    expect(mockMutate).toHaveBeenCalledWith({
      userId: 'user-2',
      role: 'viewer',
    });
  });

  it('should call mutation with contributor role when contributor is clicked', async () => {
    const user = userEvent.setup();
    renderDropdown({ currentRole: 'viewer' });
    await openSubMenu();

    await user.click(screen.getByText('Contributor'));

    expect(mockMutate).toHaveBeenCalledWith({
      userId: 'user-2',
      role: 'contributor',
    });
  });
});
