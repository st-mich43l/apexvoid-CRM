import type { AppModule, RouteDefinition } from './types'
import { NavigationRegistry } from '../navigation/registry'

export class ModuleRegistry {
  private readonly modules = new Map<string, AppModule>()

  constructor(modules: AppModule[] = []) { modules.forEach((module) => this.register(module)) }

  register(module: AppModule): void {
    if (this.modules.has(module.name)) throw new Error(`frontend module "${module.name}" is already registered`)
    this.modules.set(module.name, module)
  }

  resolve(): AppModule[] {
    const state = new Map<string, 'visiting' | 'visited'>()
    const result: AppModule[] = []
    const visit = (name: string, path: string[]): void => {
      const current = state.get(name)
      if (current === 'visited') return
      if (current === 'visiting') throw new Error(`frontend module dependency cycle: ${[...path, name].join(' -> ')}`)
      const module = this.modules.get(name)
      if (!module) throw new Error(`frontend module dependency missing: ${path.at(-1)} requires ${name}`)
      state.set(name, 'visiting')
      ;[...(module.dependencies ?? [])].sort().forEach((dependency) => visit(dependency, [...path, name]))
      state.set(name, 'visited')
      result.push(module)
    }
    ;[...this.modules.keys()].sort().forEach((name) => visit(name, []))
    return result
  }

  routes(): RouteDefinition[] { return this.resolve().flatMap((module) => module.routes ?? []) }
  navigation(): NavigationRegistry { return new NavigationRegistry(this.resolve().flatMap((module) => module.navigation ?? [])) }
}
