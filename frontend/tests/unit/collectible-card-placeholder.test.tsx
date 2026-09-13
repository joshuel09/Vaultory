import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CollectibleCard } from '@/components/collection/CollectibleCard'
import { aCollectible } from './fixtures'

describe('a collectible with no photograph', () => {
  // FR-033: a designed placeholder, not a broken image.
  it('shows the designed placeholder instead of an image element', () => {
    render(<CollectibleCard collectible={aCollectible({ image: null })} />)
    expect(screen.getByTestId('image-placeholder')).toBeInTheDocument()
    // No <img> at all, so there is nothing that can render as a broken-image icon.
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('still shows the name and status, so the card is not degraded', () => {
    render(<CollectibleCard collectible={aCollectible({ name: 'No Photo Yet' })} />)
    expect(screen.getByText('No Photo Yet')).toBeInTheDocument()
    expect(screen.getByText('Owned')).toBeInTheDocument()
  })

  // Consistency with the rest of the gallery: the placeholder occupies the same frame, so a mixed
  // collection stays on its grid.
  it('occupies the same 4:5 frame as a photograph', () => {
    const { container } = render(<CollectibleCard collectible={aCollectible({ image: null })} />)
    expect(container.querySelector('.aspect-collectible')).not.toBeNull()
  })

  it('reads as intentional rather than missing', () => {
    render(<CollectibleCard collectible={aCollectible()} />)
    expect(screen.getByText(/no photo yet/i)).toBeInTheDocument()
  })
})
