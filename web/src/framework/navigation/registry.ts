import type { NavigationItem } from '../module/types'

export class NavigationRegistry {
  private readonly itemsByID = new Map<string, NavigationItem>()

  constructor(items: NavigationItem[] = []) { items.forEach((item) => this.register(item)) }

  register(item: NavigationItem): void {
    if (this.itemsByID.has(item.id)) throw new Error(`navigation item "${item.id}" is already registered`)
    this.itemsByID.set(item.id, item)
  }

  list(): NavigationItem[] { return [...this.itemsByID.values()].sort((a, b) => a.order - b.order || a.id.localeCompare(b.id)) }
}
