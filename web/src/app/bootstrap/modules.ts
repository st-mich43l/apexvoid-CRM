import { ModuleRegistry } from '../../framework/module/registry'
import { coreModule } from '../../modules/core'
import { adminModule } from '../../modules/admin'
import { contactsModule } from '../../modules/contacts'
import { crmModule } from '../../modules/crm'
import { erpModule } from '../../modules/erp'

export const appModules = new ModuleRegistry([coreModule, adminModule, contactsModule, crmModule, erpModule])
