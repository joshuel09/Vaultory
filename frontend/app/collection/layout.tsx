import { VaultHeader } from '@/components/collection/VaultHeader'

/**
 * Wraps every vault page, so the way back out exists on all of them rather than being remembered
 * page by page.
 */
export default function CollectionLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-dvh bg-surface">
      <VaultHeader />
      {children}
    </div>
  )
}
