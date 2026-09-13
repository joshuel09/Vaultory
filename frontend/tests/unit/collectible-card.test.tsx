import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CollectibleCard } from '@/components/collection/CollectibleCard'
import { aCollectible, anImage } from './fixtures'

describe('CollectibleCard', () => {
  // FR-032: name and status are both readable on every card.
  it('shows the name and the status', () => {
    render(<CollectibleCard collectible={aCollectible({ name: 'Kaiju Sentinel' })} />)
    expect(screen.getByText('Kaiju Sentinel')).toBeInTheDocument()
    expect(screen.getByText('Owned')).toBeInTheDocument()
  })

  // FR-030, FR-014: the image is the dominant element, in the fixed gallery frame.
  it('renders the image in the 4:5 gallery frame', () => {
    const { container } = render(<CollectibleCard collectible={aCollectible({ image: anImage })} />)
    const img = screen.getByRole('img')
    expect(img).toHaveAttribute('src', anImage.renditionUrl)
    expect(img).toHaveAttribute('width', '800')
    expect(img).toHaveAttribute('height', '1000')
    expect(container.querySelector('.aspect-collectible')).not.toBeNull()
  })

  // FR-045: imagery carries a text alternative, and it names the collectible rather than saying
  // "image".
  it('gives the image a meaningful text alternative', () => {
    render(<CollectibleCard collectible={aCollectible({ name: 'Ámbar Guardián', image: anImage })} />)
    expect(screen.getByAltText(/Ámbar Guardián/)).toBeInTheDocument()
  })

  // FR-044: status must not depend on colour alone. The label itself is text, and a glyph gives a
  // second non-colour cue.
  it('conveys status without relying on colour', () => {
    render(<CollectibleCard collectible={aCollectible({ collectionStatus: 'preordered' })} />)
    const badge = screen.getByText('Preordered')
    expect(badge).toBeInTheDocument()
    expect(badge.textContent).toMatch(/Preordered/)
  })

  it('shows the series when one is recorded', () => {
    render(<CollectibleCard collectible={aCollectible({ series: 'Kaiju Wars' })} />)
    expect(screen.getByText('Kaiju Wars')).toBeInTheDocument()
  })

  it('does not invent a subtitle when nothing is recorded', () => {
    render(<CollectibleCard collectible={aCollectible()} />)
    expect(screen.queryByText('null')).not.toBeInTheDocument()
    expect(screen.queryByText('undefined')).not.toBeInTheDocument()
  })
})
