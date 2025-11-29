import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { EnergyField } from './EnergyField'
import { useRecordContext, useDataProvider, useNotify } from 'react-admin'
import httpClient from '../dataProvider/httpClient'

// Mock react-admin hooks
vi.mock('react-admin', () => ({
  useRecordContext: vi.fn(),
  useDataProvider: vi.fn(),
  useNotify: vi.fn(),
}))

// Mock httpClient
vi.mock('../dataProvider/httpClient', () => ({
  default: vi.fn(),
}))

describe('EnergyField', () => {
  const mockDataProvider = {
    getOne: vi.fn().mockResolvedValue({ data: {} }),
  }
  const mockNotify = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    useDataProvider.mockReturnValue(mockDataProvider)
    useNotify.mockReturnValue(mockNotify)
  })

  describe('display', () => {
    it('renders three squares for energy levels', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })

      render(<EnergyField resource="song" />)

      // Should have three squares with titles
      expect(screen.getByTitle('Low')).toBeInTheDocument()
      expect(screen.getByTitle('Medium')).toBeInTheDocument()
      expect(screen.getByTitle('High')).toBeInTheDocument()
    })

    it('highlights the selected energy level', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: { energy: ['medium'] },
      })

      const { container } = render(<EnergyField resource="song" />)

      // The medium square should have the 'selected' class
      const squares = container.querySelectorAll('[class*="square"]')
      expect(squares).toHaveLength(3)

      // Medium is the second square (index 1)
      expect(squares[1].className).toContain('selected')
    })

    it('shows all squares as unselected when no energy is set', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })

      const { container } = render(<EnergyField resource="song" />)

      const squares = container.querySelectorAll('[class*="square"]')
      squares.forEach((square) => {
        expect(square.className).toContain('unselected')
      })
    })
  })

  describe('interaction', () => {
    it('calls API when clicking a square', async () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })
      httpClient.mockResolvedValue({ json: {} })

      render(<EnergyField resource="song" />)

      // Click the high square
      fireEvent.click(screen.getByTitle('High'))

      await waitFor(() => {
        expect(httpClient).toHaveBeenCalledWith(
          '/api/song/song-1/energy',
          expect.objectContaining({
            method: 'PUT',
            body: JSON.stringify({ value: 'high' }),
          }),
        )
      })
    })

    it('unsets energy when clicking the currently selected square', async () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: { energy: ['high'] },
      })
      httpClient.mockResolvedValue({ json: {} })

      render(<EnergyField resource="song" />)

      // Click the already-selected high square
      fireEvent.click(screen.getByTitle('High'))

      await waitFor(() => {
        expect(httpClient).toHaveBeenCalledWith(
          '/api/song/song-1/energy',
          expect.objectContaining({
            method: 'PUT',
            body: JSON.stringify({ value: '' }),
          }),
        )
      })
    })

    it('is disabled when track is missing', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
        missing: true,
      })

      const { container } = render(<EnergyField resource="song" />)

      // Container should have disabled class
      expect(container.firstChild.className).toContain('disabled')
    })

    it('refreshes record after successful update', async () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })
      httpClient.mockResolvedValue({ json: {} })

      render(<EnergyField resource="song" />)

      fireEvent.click(screen.getByTitle('Medium'))

      await waitFor(() => {
        expect(mockDataProvider.getOne).toHaveBeenCalledWith('song', {
          id: 'song-1',
        })
      })
    })

    it('shows error notification on API failure', async () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })
      httpClient.mockRejectedValue(new Error('Network error'))

      render(<EnergyField resource="song" />)

      fireEvent.click(screen.getByTitle('Low'))

      await waitFor(() => {
        expect(mockNotify).toHaveBeenCalledWith('ra.page.error', 'warning')
      })
    })
  })

  describe('playlist tracks', () => {
    it('uses mediaFileId for playlist tracks', async () => {
      useRecordContext.mockReturnValue({
        id: 'playlist-track-1',
        mediaFileId: 'song-1',
        playlistId: 'playlist-1',
        tags: {},
      })
      httpClient.mockResolvedValue({ json: {} })

      render(<EnergyField resource="song" />)

      fireEvent.click(screen.getByTitle('High'))

      await waitFor(() => {
        expect(httpClient).toHaveBeenCalledWith(
          '/api/song/song-1/energy',
          expect.objectContaining({
            method: 'PUT',
          }),
        )
      })
    })
  })
})
