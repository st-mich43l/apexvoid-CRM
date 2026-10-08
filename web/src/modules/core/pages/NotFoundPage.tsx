import { Link } from 'react-router-dom'
import { Button, EmptyState, PageContainer, PageHeader } from '../../../components/ui'

export function NotFoundPage() { return <PageContainer variant="detail"><PageHeader eyebrow="ApexVoid Framework" title="Page not found" description="The requested route is not registered by an application module." /><div className="rounded-xl border border-border bg-card"><EmptyState title="This page does not exist" description="Return to your workspace and choose a destination from the navigation." action={<Link to="/"><Button>Back to dashboard</Button></Link>} /></div></PageContainer> }
