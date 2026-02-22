export type FileItemRef = {
  id: string;
  name: string;
  type: 'file' | 'folder';
};

export type FilePreviewRef = {
  id: string;
  name: string;
  mimeType: string;
  size: number;
};
