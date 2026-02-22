import {
  authKeys,
  folderKeys,
  fileKeys,
  trashKeys,
  groupKeys,
  uploadKeys,
  shareKeys,
  profileKeys,
} from '@/lib/api/queries';

describe('authKeys', () => {
  it('all returns ["auth"]', () => {
    expect(authKeys.all).toEqual(['auth']);
  });

  it('me() returns ["auth", "me"]', () => {
    expect(authKeys.me()).toEqual(['auth', 'me']);
  });
});

describe('folderKeys', () => {
  it('all returns ["folders"]', () => {
    expect(folderKeys.all).toEqual(['folders']);
  });

  it('lists() returns ["folders", "list"]', () => {
    expect(folderKeys.lists()).toEqual(['folders', 'list']);
  });

  it('contents() with folderId returns ["folders", "list", folderId]', () => {
    expect(folderKeys.contents('abc')).toEqual(['folders', 'list', 'abc']);
  });

  it('contents() with null returns ["folders", "list", "root"]', () => {
    expect(folderKeys.contents(null)).toEqual(['folders', 'list', 'root']);
  });

  it('details() returns ["folders", "detail"]', () => {
    expect(folderKeys.details()).toEqual(['folders', 'detail']);
  });

  it('detail() returns ["folders", "detail", id]', () => {
    expect(folderKeys.detail('xyz')).toEqual(['folders', 'detail', 'xyz']);
  });

  it('ancestors() returns ["folders", "ancestors", id]', () => {
    expect(folderKeys.ancestors('xyz')).toEqual([
      'folders',
      'ancestors',
      'xyz',
    ]);
  });
});

describe('fileKeys', () => {
  it('all returns ["files"]', () => {
    expect(fileKeys.all).toEqual(['files']);
  });

  it('versions() returns ["files", "versions", fileId]', () => {
    expect(fileKeys.versions('f1')).toEqual(['files', 'versions', 'f1']);
  });

  it('download() returns ["files", "download", fileId]', () => {
    expect(fileKeys.download('f1')).toEqual(['files', 'download', 'f1']);
  });
});

describe('trashKeys', () => {
  it('all returns ["trash"]', () => {
    expect(trashKeys.all).toEqual(['trash']);
  });

  it('lists() returns ["trash", "list"]', () => {
    expect(trashKeys.lists()).toEqual(['trash', 'list']);
  });

  it('list() returns ["trash", "list", { limit, cursor }]', () => {
    expect(trashKeys.list()).toEqual([
      'trash',
      'list',
      { limit: undefined, cursor: undefined },
    ]);
  });

  it('list(50, "abc") returns ["trash", "list", { limit: 50, cursor: "abc" }]', () => {
    expect(trashKeys.list(50, 'abc')).toEqual([
      'trash',
      'list',
      { limit: 50, cursor: 'abc' },
    ]);
  });
});

describe('groupKeys', () => {
  it('all returns ["groups"]', () => {
    expect(groupKeys.all).toEqual(['groups']);
  });

  it('lists() returns ["groups", "list"]', () => {
    expect(groupKeys.lists()).toEqual(['groups', 'list']);
  });

  it('details() returns ["groups", "detail"]', () => {
    expect(groupKeys.details()).toEqual(['groups', 'detail']);
  });

  it('detail() returns ["groups", "detail", id]', () => {
    expect(groupKeys.detail('g1')).toEqual(['groups', 'detail', 'g1']);
  });

  it('members() returns ["groups", "members", groupId]', () => {
    expect(groupKeys.members('g1')).toEqual(['groups', 'members', 'g1']);
  });

  it('invitations() returns ["groups", "invitations", groupId]', () => {
    expect(groupKeys.invitations('g1')).toEqual([
      'groups',
      'invitations',
      'g1',
    ]);
  });

  it('pending() returns ["groups", "pending"]', () => {
    expect(groupKeys.pending()).toEqual(['groups', 'pending']);
  });
});

describe('uploadKeys', () => {
  it('all returns ["uploads"]', () => {
    expect(uploadKeys.all).toEqual(['uploads']);
  });

  it('status() returns ["uploads", "status", sessionId]', () => {
    expect(uploadKeys.status('s1')).toEqual(['uploads', 'status', 's1']);
  });
});

describe('shareKeys', () => {
  it('all returns ["shares"]', () => {
    expect(shareKeys.all).toEqual(['shares']);
  });

  it('list() returns ["shares", resourceType, resourceId]', () => {
    expect(shareKeys.list('folder', 'f1')).toEqual(['shares', 'folder', 'f1']);
  });
});

describe('profileKeys', () => {
  it('all returns ["profile"]', () => {
    expect(profileKeys.all).toEqual(['profile']);
  });
});
