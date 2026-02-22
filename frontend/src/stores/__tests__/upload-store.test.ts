import { useUploadStore } from '@/stores/upload-store';

describe('useUploadStore', () => {
  beforeEach(() => {
    useUploadStore.setState({ uploads: new Map() });
  });

  it('has empty uploads initially', () => {
    expect(useUploadStore.getState().uploads.size).toBe(0);
  });

  it('addUpload creates item with progress=0 and status=pending', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'file.txt', fileSize: 1024 });

    const item = useUploadStore.getState().uploads.get('u1');
    expect(item).toBeDefined();
    expect(item!.progress).toBe(0);
    expect(item!.status).toBe('pending');
    expect(item!.fileName).toBe('file.txt');
    expect(item!.fileSize).toBe(1024);
  });

  it('updateProgress changes progress and sets status to uploading', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'file.txt', fileSize: 1024 });

    useUploadStore.getState().updateProgress('u1', 50);

    const item = useUploadStore.getState().uploads.get('u1');
    expect(item!.progress).toBe(50);
    expect(item!.status).toBe('uploading');
  });

  it('updateProgress does nothing for non-existent id', () => {
    useUploadStore.getState().updateProgress('non-existent', 50);
    expect(useUploadStore.getState().uploads.size).toBe(0);
  });

  it('setStatus with completed sets progress to 100', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'file.txt', fileSize: 1024 });
    useUploadStore.getState().updateProgress('u1', 50);

    useUploadStore.getState().setStatus('u1', 'completed');

    const item = useUploadStore.getState().uploads.get('u1');
    expect(item!.status).toBe('completed');
    expect(item!.progress).toBe(100);
  });

  it('setStatus with failed keeps current progress and stores error', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'file.txt', fileSize: 1024 });
    useUploadStore.getState().updateProgress('u1', 30);

    useUploadStore.getState().setStatus('u1', 'failed', 'Network error');

    const item = useUploadStore.getState().uploads.get('u1');
    expect(item!.status).toBe('failed');
    expect(item!.progress).toBe(30);
    expect(item!.error).toBe('Network error');
  });

  it('removeUpload deletes item', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'file.txt', fileSize: 1024 });

    useUploadStore.getState().removeUpload('u1');

    expect(useUploadStore.getState().uploads.has('u1')).toBe(false);
  });

  it('clearCompleted removes only completed items', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'a.txt', fileSize: 100 });
    useUploadStore
      .getState()
      .addUpload({ id: 'u2', fileName: 'b.txt', fileSize: 200 });
    useUploadStore
      .getState()
      .addUpload({ id: 'u3', fileName: 'c.txt', fileSize: 300 });

    useUploadStore.getState().setStatus('u1', 'completed');
    useUploadStore.getState().setStatus('u3', 'completed');

    useUploadStore.getState().clearCompleted();

    const uploads = useUploadStore.getState().uploads;
    expect(uploads.size).toBe(1);
    expect(uploads.has('u2')).toBe(true);
    expect(uploads.has('u1')).toBe(false);
    expect(uploads.has('u3')).toBe(false);
  });

  it('activeCount counts pending and uploading items', () => {
    useUploadStore
      .getState()
      .addUpload({ id: 'u1', fileName: 'a.txt', fileSize: 100 });
    useUploadStore
      .getState()
      .addUpload({ id: 'u2', fileName: 'b.txt', fileSize: 200 });
    useUploadStore
      .getState()
      .addUpload({ id: 'u3', fileName: 'c.txt', fileSize: 300 });

    // u1: pending, u2: uploading, u3: completed
    useUploadStore.getState().updateProgress('u2', 50);
    useUploadStore.getState().setStatus('u3', 'completed');

    expect(useUploadStore.getState().activeCount()).toBe(2);
  });
});
