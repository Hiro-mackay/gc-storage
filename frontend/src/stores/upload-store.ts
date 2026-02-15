import { create } from 'zustand'

type UploadStatus = 'pending' | 'uploading' | 'completed' | 'failed'

interface UploadItem {
  id: string
  fileName: string
  fileSize: number
  progress: number
  status: UploadStatus
  error?: string
}

interface UploadState {
  uploads: Map<string, UploadItem>
  addUpload: (item: Omit<UploadItem, 'progress' | 'status'>) => void
  updateProgress: (id: string, progress: number) => void
  setStatus: (id: string, status: UploadStatus, error?: string) => void
  removeUpload: (id: string) => void
  clearCompleted: () => void
  activeCount: () => number
}

export const useUploadStore = create<UploadState>((set, get) => ({
  uploads: new Map(),
  addUpload: (item) =>
    set((state) => {
      const next = new Map(state.uploads)
      next.set(item.id, { ...item, progress: 0, status: 'pending' })
      return { uploads: next }
    }),
  updateProgress: (id, progress) =>
    set((state) => {
      const next = new Map(state.uploads)
      const item = next.get(id)
      if (item) {
        next.set(id, { ...item, progress, status: 'uploading' })
      }
      return { uploads: next }
    }),
  setStatus: (id, status, error) =>
    set((state) => {
      const next = new Map(state.uploads)
      const item = next.get(id)
      if (item) {
        next.set(id, { ...item, status, error, progress: status === 'completed' ? 100 : item.progress })
      }
      return { uploads: next }
    }),
  removeUpload: (id) =>
    set((state) => {
      const next = new Map(state.uploads)
      next.delete(id)
      return { uploads: next }
    }),
  clearCompleted: () =>
    set((state) => {
      const next = new Map(state.uploads)
      for (const [id, item] of next) {
        if (item.status === 'completed') next.delete(id)
      }
      return { uploads: next }
    }),
  activeCount: () => {
    let count = 0
    for (const item of get().uploads.values()) {
      if (item.status === 'pending' || item.status === 'uploading') count++
    }
    return count
  },
}))
