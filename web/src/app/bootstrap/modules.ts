import { ModuleRegistry } from '../../framework/module/registry'
import { coreModule } from '../../modules/core'

export const appModules = new ModuleRegistry([coreModule])
