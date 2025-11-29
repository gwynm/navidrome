import React, { useState, useCallback } from 'react'
import PropTypes from 'prop-types'
import { useDataProvider, useNotify, useRecordContext } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import httpClient from '../dataProvider/httpClient'

const ENERGY_VALUES = ['low', 'medium', 'high']

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
  squareLow: {
    backgroundColor: theme.palette.success.main,
  },
  squareMedium: {
    backgroundColor: theme.palette.warning.main,
  },
  squareHigh: {
    backgroundColor: theme.palette.error.main,
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

export const EnergyField = ({ resource, ...rest }) => {
  const record = useRecordContext(rest) || {}
  const classes = useStyles()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [loading, setLoading] = useState(false)

  // Get current energy value from tags
  const currentEnergy = record?.tags?.energy?.[0] || ''

  const setEnergy = useCallback(
    async (value) => {
      const id = record.mediaFileId || record.id
      setLoading(true)

      try {
        await httpClient(`/api/song/${id}/energy`, {
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
      } catch (e) {
        console.error('Error setting energy:', e)
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
      const newValue = value === currentEnergy ? '' : value
      setEnergy(newValue)
    },
    [currentEnergy, setEnergy, record?.missing, loading],
  )

  const isDisabled = record?.missing
  const containerClass = `${classes.container} ${isDisabled ? classes.disabled : ''} ${loading ? classes.loading : ''}`

  return (
    <div className={containerClass} onClick={(e) => e.stopPropagation()}>
      {ENERGY_VALUES.map((value) => {
        const isSelected = value === currentEnergy
        const squareColorClass =
          value === 'low'
            ? classes.squareLow
            : value === 'medium'
              ? classes.squareMedium
              : classes.squareHigh
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

EnergyField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object,
}

export default EnergyField
