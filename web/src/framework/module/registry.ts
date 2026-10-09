import type { ApplicationMetadata } from '../metadata/types'
import type { AppModule, FrontendApplication, RouteDefinition } from './types'
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

  applications(): FrontendApplication[] {
    const applications = this.resolve().flatMap(module => module.application ? [module.application] : [])
    const routes = this.routes()
    const navigation = this.navigation().list()
    const seen = new Set<string>()
    for (const application of applications) {
      if (seen.has(application.id)) throw new Error(`frontend application "${application.id}" is already registered`)
      seen.add(application.id)
      if (!routes.some(route => route.path === application.entryRoute)) throw new Error(`frontend application "${application.id}" entry route "${application.entryRoute}" is not registered`)
      if (!navigation.some(item => item.id === application.navigationID)) throw new Error(`frontend application "${application.id}" navigation "${application.navigationID}" is not registered`)
    }
    return [...applications].sort((a, b) => a.id.localeCompare(b.id))
  }

  applicationContracts(registered: ApplicationMetadata[]): ApplicationContract[] {
    const compiled = new Map(this.applications().map(application => [application.id, application]))
    const backend = new Map(registered.map(application => [application.id, application]))
    const ids = [...new Set([...compiled.keys(), ...backend.keys()])].sort()
    return ids.map(id => {
      const frontend = compiled.get(id)
      const application = backend.get(id)
      if (!application) return { id, frontend, status: 'missing-backend' as const, message: `This frontend includes ${id}, but the connected backend did not register it.` }
      if (!frontend) return { id, application, status: 'missing-frontend' as const, message: `The backend registered ${id}, but this frontend build has no compiled entry for it.` }
      if (application.frontend.entry_route !== frontend.entryRoute) return { id, application, frontend, status: 'route-mismatch' as const, message: `The compiled frontend entry route does not match the backend registration for ${id}.` }
      if (application.frontend.navigation_id !== frontend.navigationID) return { id, application, frontend, status: 'navigation-mismatch' as const, message: `The compiled frontend navigation identity does not match the backend registration for ${id}.` }
      if (application.api_contract_version !== frontend.apiContractVersion) return { id, application, frontend, status: 'api-version-mismatch' as const, message: `The compiled frontend API contract version does not match the backend registration for ${id}.` }
      return { id, application, frontend, status: 'compatible' as const, message: 'Registration metadata and declared API contract version agree.' }
    })
  }
}

export type ApplicationContract = { id: string; application?: ApplicationMetadata; frontend?: FrontendApplication; status: 'compatible' | 'missing-backend' | 'missing-frontend' | 'route-mismatch' | 'navigation-mismatch' | 'api-version-mismatch'; message: string }
