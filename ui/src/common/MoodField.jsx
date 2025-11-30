import React, { useState, useCallback, useMemo } from 'react'
import PropTypes from 'prop-types'
import { useDataProvider, useNotify, useRecordContext } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import httpClient from '../dataProvider/httpClient'

const MOOD_VALUES = ['negative', 'neutral', 'positive']
const EMPTY_RECORD = {}

const useStyles = makeStyles((theme) => ({
  container: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '3px',
    padding: '2px 0',
  },
  square: {
    width: '12px',
    height: '12px',
    borderRadius: '2px',
    cursor: 'pointer',
    transition: 'transform 0.1s, opacity 0.1s',
    '&:hover': {
      transform: 'scale(1.2)',
    },
  },
  squareNegative: {
    backgroundColor: '#4C0763',
  },
  squareNeutral: {
    backgroundColor: '#57C785',
  },
  squarePositive: {
    backgroundColor: '#FF00EA',
  },
  // Unselected squares are dimmed
  unselected: {
    opacity: 0.25,
    '&:hover': {
      opacity: 0.6,
    },
  },
  // Selected square is fully visible
  selected: {
    opacity: 1,
    boxShadow: `0 0 0 1px ${theme.palette.background.paper}, 0 0 0 2px currentColor`,
  },
  disabled: {
    opacity: 0.3,
    pointerEvents: 'none',
  },
  loading: {
    opacity: 0.5,
    pointerEvents: 'none',
  },
}))

export const MoodField = ({ resource, ...rest }) => {
  const contextRecord = useRecordContext(rest)
  const record = useMemo(() => contextRecord || EMPTY_RECORD, [contextRecord])
  const classes = useStyles()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [loading, setLoading] = useState(false)

  // Get current mood value from tags
  const currentMood = record?.tags?.mood?.[0] || ''

  const setMood = useCallback(
    async (value) => {
      const id = record.mediaFileId || record.id
      setLoading(true)

      try {
        await httpClient(`/api/song/${id}/mood`, {
          method: 'PUT',
          body: JSON.stringify({ value }),
        })

        // Refresh the record to get updated data
        if (record.mediaFileId) {
          // Playlist track - refresh both
          await Promise.all([
            dataProvider.getOne('song', { id: record.mediaFileId }),
            dataProvider.getOne('playlistTrack', {
              id: record.id,
              filter: { playlist_id: record.playlistId },
            }),
          ])
        } else {
          await dataProvider.getOne(resource, { id: record.id })
        }
      } catch {
        notify('ra.page.error', 'warning')
      } finally {
        setLoading(false)
      }
    },
    [record, dataProvider, resource, notify],
  )

  const handleSquareClick = useCallback(
    (e, value) => {
      e.stopPropagation()
      if (record?.missing || loading) return
      // If clicking the current value, unset it
      const newValue = value === currentMood ? '' : value
      setMood(newValue)
    },
    [currentMood, setMood, record?.missing, loading],
  )

  const isDisabled = record?.missing
  const containerClass = `${classes.container} ${isDisabled ? classes.disabled : ''} ${loading ? classes.loading : ''}`

  return (
    <div className={containerClass} onClick={(e) => e.stopPropagation()}>
      {MOOD_VALUES.map((value) => {
        const isSelected = value === currentMood
        const squareColorClass =
          value === 'negative'
            ? classes.squareNegative
            : value === 'neutral'
              ? classes.squareNeutral
              : classes.squarePositive
        const selectionClass = isSelected
          ? classes.selected
          : classes.unselected

        return (
          <div
            key={value}
            className={`${classes.square} ${squareColorClass} ${selectionClass}`}
            onClick={(e) => handleSquareClick(e, value)}
            title={value.charAt(0).toUpperCase() + value.slice(1)}
          />
        )
      })}
    </div>
  )
}

MoodField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object,
}

export default MoodField
