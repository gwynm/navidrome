import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MoodField } from './MoodField'
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

describe('MoodField', () => {
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
    it('renders three squares for mood levels', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })

      render(<MoodField resource="song" />)

      // Should have three squares with titles
      expect(screen.getByTitle('Negative')).toBeInTheDocument()
      expect(screen.getByTitle('Neutral')).toBeInTheDocument()
      expect(screen.getByTitle('Positive')).toBeInTheDocument()
    })

    it('highlights the selected mood level', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: { mood: ['neutral'] },
      })

      const { container } = render(<MoodField resource="song" />)

      // The neutral square should have the 'selected' class
      const squares = container.querySelectorAll('[class*="square"]')
      expect(squares).toHaveLength(3)

      // Neutral is the second square (index 1)
      expect(squares[1].className).toContain('selected')
    })

    it('shows all squares as unselected when no mood is set', () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })

      const { container } = render(<MoodField resource="song" />)

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

      render(<MoodField resource="song" />)

      // Click the positive square
      fireEvent.click(screen.getByTitle('Positive'))

      await waitFor(() => {
        expect(httpClient).toHaveBeenCalledWith(
          '/api/song/song-1/mood',
          expect.objectContaining({
            method: 'PUT',
            body: JSON.stringify({ value: 'positive' }),
          }),
        )
      })
    })

    it('unsets mood when clicking the currently selected square', async () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: { mood: ['positive'] },
      })
      httpClient.mockResolvedValue({ json: {} })

      render(<MoodField resource="song" />)

      // Click the already-selected positive square
      fireEvent.click(screen.getByTitle('Positive'))

      await waitFor(() => {
        expect(httpClient).toHaveBeenCalledWith(
          '/api/song/song-1/mood',
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

      const { container } = render(<MoodField resource="song" />)

      // Container should have disabled class
      expect(container.firstChild.className).toContain('disabled')
    })

    it('refreshes record after successful update', async () => {
      useRecordContext.mockReturnValue({
        id: 'song-1',
        tags: {},
      })
      httpClient.mockResolvedValue({ json: {} })

      render(<MoodField resource="song" />)

      fireEvent.click(screen.getByTitle('Neutral'))

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

      render(<MoodField resource="song" />)

      fireEvent.click(screen.getByTitle('Negative'))

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

      render(<MoodField resource="song" />)

      fireEvent.click(screen.getByTitle('Positive'))

      await waitFor(() => {
        expect(httpClient).toHaveBeenCalledWith(
          '/api/song/song-1/mood',
          expect.objectContaining({
            method: 'PUT',
          }),
        )
      })
    })
  })
})
