import { render, screen } from '@testing-library/react';
import { createWrapper } from '@/test/test-utils';
import { SharedFolderBrowser } from '../shared-folder-browser';

const fileItem = {
  id: 'file-1',
  name: 'report.pdf',
  type: 'file' as const,
  size: 1024,
};
const folderItem = { id: 'folder-1', name: 'Photos', type: 'folder' as const };

function renderComponent(
  contents: Parameters<typeof SharedFolderBrowser>[0]['contents'] = [],
  token = 'test-token',
) {
  return render(
    <SharedFolderBrowser contents={contents} permission="read" token={token} />,
    { wrapper: createWrapper() },
  );
}

describe('SharedFolderBrowser', () => {
  it('should render empty state when contents is empty', () => {
    renderComponent([]);
    expect(screen.getByText(/this folder is empty/i)).toBeInTheDocument();
  });

  it('should render file item name', () => {
    renderComponent([fileItem]);
    expect(screen.getByText('report.pdf')).toBeInTheDocument();
  });

  it('should render folder item name', () => {
    renderComponent([folderItem]);
    expect(screen.getByText('Photos')).toBeInTheDocument();
  });

  it('should show download link for file items', () => {
    renderComponent([fileItem], 'abc123');
    const downloadLink = screen.getByRole('link', {
      name: /download report.pdf/i,
    });
    expect(downloadLink).toBeInTheDocument();
    expect(downloadLink).toHaveAttribute(
      'href',
      '/api/v1/share/abc123/download?fileId=file-1',
    );
  });

  it('should not show download link for folder items', () => {
    renderComponent([folderItem]);
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('should display file size when provided', () => {
    renderComponent([fileItem]);
    expect(screen.getByText('1 KB')).toBeInTheDocument();
  });

  it('should render multiple items', () => {
    renderComponent([fileItem, folderItem]);
    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    expect(screen.getByText('Photos')).toBeInTheDocument();
  });
});
