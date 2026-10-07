import { ModuleRegistry } from '../../framework/module/registry'
import { coreModule } from '../../modules/core'
import { adminModule } from '../../modules/admin'

export const appModules = new ModuleRegistry([coreModule, adminModule])
