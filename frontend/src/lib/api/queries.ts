export const authKeys = {
  all: ['auth'] as const,
  me: () => [...authKeys.all, 'me'] as const,
}

export const folderKeys = {
  all: ['folders'] as const,
  lists: () => [...folderKeys.all, 'list'] as const,
  contents: (folderId: string | null) => [...folderKeys.lists(), folderId ?? 'root'] as const,
  details: () => [...folderKeys.all, 'detail'] as const,
  detail: (id: string) => [...folderKeys.details(), id] as const,
  ancestors: (id: string) => [...folderKeys.all, 'ancestors', id] as const,
}

export const fileKeys = {
  all: ['files'] as const,
  versions: (fileId: string) => [...fileKeys.all, 'versions', fileId] as const,
  download: (fileId: string) => [...fileKeys.all, 'download', fileId] as const,
}

export const trashKeys = {
  all: ['trash'] as const,
  list: () => [...trashKeys.all, 'list'] as const,
}

export const groupKeys = {
  all: ['groups'] as const,
  lists: () => [...groupKeys.all, 'list'] as const,
  details: () => [...groupKeys.all, 'detail'] as const,
  detail: (id: string) => [...groupKeys.details(), id] as const,
  members: (groupId: string) => [...groupKeys.all, 'members', groupId] as const,
  invitations: (groupId: string) => [...groupKeys.all, 'invitations', groupId] as const,
  pending: () => [...groupKeys.all, 'pending'] as const,
}
