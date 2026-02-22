import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { RoleDropdown } from '../role-dropdown';

describe('RoleDropdown', () => {
  it('should render a select with role options', () => {
    render(<RoleDropdown value="viewer" onChange={() => {}} />, {
      wrapper: createWrapper(),
    });
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('should show viewer as selected when value is viewer', () => {
    render(<RoleDropdown value="viewer" onChange={() => {}} />, {
      wrapper: createWrapper(),
    });
    expect(screen.getByRole('combobox')).toHaveTextContent('Viewer');
  });

  it('should show contributor as selected when value is contributor', () => {
    render(<RoleDropdown value="contributor" onChange={() => {}} />, {
      wrapper: createWrapper(),
    });
    expect(screen.getByRole('combobox')).toHaveTextContent('Contributor');
  });

  it('should show content_manager as selected when value is content_manager', () => {
    render(<RoleDropdown value="content_manager" onChange={() => {}} />, {
      wrapper: createWrapper(),
    });
    expect(screen.getByRole('combobox')).toHaveTextContent('Content Manager');
  });

  it('should call onChange when a role is selected', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(<RoleDropdown value="viewer" onChange={handleChange} />, {
      wrapper: createWrapper(),
    });

    await user.selectOptions(screen.getByRole('combobox'), 'contributor');

    expect(handleChange).toHaveBeenCalledWith('contributor');
  });

  it('should be disabled when disabled prop is true', () => {
    render(<RoleDropdown value="viewer" onChange={() => {}} disabled />, {
      wrapper: createWrapper(),
    });
    expect(screen.getByRole('combobox')).toBeDisabled();
  });
});
