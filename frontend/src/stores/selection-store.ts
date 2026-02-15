import { create } from 'zustand'

interface SelectionState {
  selectedIds: Set<string>
  lastSelectedId: string | null
  select: (id: string) => void
  toggle: (id: string) => void
  selectRange: (ids: string[]) => void
  selectAll: (ids: string[]) => void
  clear: () => void
  isSelected: (id: string) => boolean
}

export const useSelectionStore = create<SelectionState>((set, get) => ({
  selectedIds: new Set(),
  lastSelectedId: null,
  select: (id) => set({ selectedIds: new Set([id]), lastSelectedId: id }),
  toggle: (id) =>
    set((state) => {
      const next = new Set(state.selectedIds)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return { selectedIds: next, lastSelectedId: id }
    }),
  selectRange: (ids) =>
    set((state) => {
      const next = new Set(state.selectedIds)
      ids.forEach((id) => next.add(id))
      return { selectedIds: next, lastSelectedId: ids[ids.length - 1] ?? state.lastSelectedId }
    }),
  selectAll: (ids) => set({ selectedIds: new Set(ids), lastSelectedId: null }),
  clear: () => set({ selectedIds: new Set(), lastSelectedId: null }),
  isSelected: (id) => get().selectedIds.has(id),
}))
