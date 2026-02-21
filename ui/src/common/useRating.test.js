import { renderHook, act } from '@testing-library/react-hooks'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { useRating } from './useRating'
import subsonic from '../subsonic'

vi.mock('../subsonic', () => ({
  default: {
    setRating: vi.fn(() => Promise.resolve()),
  },
}))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useNotify: vi.fn(() => vi.fn()),
  }
})

describe('useRating', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns rating value from record', () => {
    const record = { id: 'sg-1', rating: 3 }
    const { result } = renderHook(() => useRating('song', record))
    const [rate, rating, loading] = result.current
    expect(rating).toBe(3)
    expect(loading).toBe(false)
    expect(typeof rate).toBe('function')
  })

  it('sets rating using targetId and calls setRating API', async () => {
    const record = { id: 'sg-1', rating: 0 }
    const { result } = renderHook(() => useRating('song', record))
    await act(async () => {
      await result.current[0](4, 'sg-1')
    })
    expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 4)
  })

  it('handles zero rating (unrate)', async () => {
    const record = { id: 'sg-1', rating: 5 }
    const { result } = renderHook(() => useRating('song', record))
    await act(async () => {
      await result.current[0](0, 'sg-1')
    })
    expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 0)
  })

  it('calls afterRate callback when provided', async () => {
    const afterRate = vi.fn(() => Promise.resolve())
    const record = { id: 'sg-1', rating: 2 }
    const { result } = renderHook(() => useRating('song', record, afterRate))
    await act(async () => {
      await result.current[0](5, 'sg-1')
    })
    expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 5)
    expect(afterRate).toHaveBeenCalledWith(5)
  })

  it('updates local rating optimistically', async () => {
    const record = { id: 'sg-1', rating: 2 }
    const { result } = renderHook(() => useRating('song', record))
    await act(async () => {
      await result.current[0](4, 'sg-1')
    })
    expect(result.current[1]).toBe(4)
  })

  it('reverts local rating on error', async () => {
    subsonic.setRating.mockRejectedValueOnce(new Error('fail'))
    const record = { id: 'sg-1', rating: 2 }
    const { result } = renderHook(() => useRating('song', record))
    await act(async () => {
      await result.current[0](4, 'sg-1')
    })
    expect(result.current[1]).toBe(2)
  })
})
