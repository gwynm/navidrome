import React, { useState, useCallback, useEffect, useMemo } from 'react'
import PropTypes from 'prop-types'
import {
  useDataProvider,
  useNotify,
  useRecordContext,
  useRefresh,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { Typography } from '@material-ui/core'
import httpClient from '../dataProvider/httpClient'

const ENERGY_VALUES = ['low', 'medium', 'high']
const MOOD_VALUES = ['negative', 'neutral', 'positive']

const useStyles = makeStyles((theme) => ({
  container: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(2),
    marginTop: theme.spacing(0.5),
  },
  fieldGroup: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '3px',
  },
  label: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
    marginRight: theme.spacing(0.5),
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
  // Energy colors (from theme)
  squareEnergyLow: {
    backgroundColor: theme.palette.success.main,
  },
  squareEnergyMedium: {
    backgroundColor: theme.palette.warning.main,
  },
  squareEnergyHigh: {
    backgroundColor: theme.palette.error.main,
  },
  // Mood colors (custom)
  squareMoodNegative: {
    backgroundColor: '#4C0763',
  },
  squareMoodNeutral: {
    backgroundColor: '#57C785',
  },
  squareMoodPositive: {
    backgroundColor: '#FF00EA',
  },
  unselected: {
    opacity: 0.25,
    '&:hover': {
      opacity: 0.6,
    },
  },
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

// Get the common value for a tag across all songs (or '' if mixed)
const getCommonTagValue = (songs, tagName) => {
  if (!songs || songs.length === 0) return ''
  const firstValue = songs[0]?.tags?.[tagName]?.[0] || ''
  const allSame = songs.every(
    (song) => (song?.tags?.[tagName]?.[0] || '') === firstValue,
  )
  return allSame ? firstValue : ''
}

const AlbumTagFields = ({ size }) => {
  const record = useRecordContext()
  const classes = useStyles()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const refresh = useRefresh()
  const [songs, setSongs] = useState([])
  const [loading, setLoading] = useState(false)
  const [songsLoaded, setSongsLoaded] = useState(false)

  // Fetch songs for this album
  useEffect(() => {
    if (!record?.id) return

    dataProvider
      .getList('song', {
        filter: { album_id: record.id },
        pagination: { page: 1, perPage: 1000 },
        sort: { field: 'album', order: 'ASC' },
      })
      .then(({ data }) => {
        setSongs(data || [])
        setSongsLoaded(true)
      })
      .catch(() => {
        setSongsLoaded(true)
      })
  }, [record?.id, dataProvider])

  const currentEnergy = useMemo(
    () => getCommonTagValue(songs, 'energy'),
    [songs],
  )
  const currentMood = useMemo(() => getCommonTagValue(songs, 'mood'), [songs])

  const setTagForAllSongs = useCallback(
    async (tagName, value) => {
      if (songs.length === 0) return
      setLoading(true)

      try {
        // Update all songs in parallel
        await Promise.all(
          songs.map((song) =>
            httpClient(`/api/song/${song.id}/${tagName}`, {
              method: 'PUT',
              body: JSON.stringify({ value }),
            }),
          ),
        )
        // Refresh to get updated data
        refresh()
      } catch {
        notify('ra.page.error', 'warning')
      } finally {
        setLoading(false)
      }
    },
    [songs, notify, refresh],
  )

  const handleEnergyClick = useCallback(
    (e, value) => {
      e.stopPropagation()
      if (loading) return
      const newValue = value === currentEnergy ? '' : value
      setTagForAllSongs('energy', newValue)
    },
    [currentEnergy, setTagForAllSongs, loading],
  )

  const handleMoodClick = useCallback(
    (e, value) => {
      e.stopPropagation()
      if (loading) return
      const newValue = value === currentMood ? '' : value
      setTagForAllSongs('mood', newValue)
    },
    [currentMood, setTagForAllSongs, loading],
  )

  // Don't render until songs are loaded
  if (!songsLoaded || songs.length === 0) return null

  const containerClass = `${classes.container} ${loading ? classes.loading : ''}`
  const isSmall = size === 'small'

  return (
    <div className={containerClass} onClick={(e) => e.stopPropagation()}>
      {/* Energy */}
      <div className={classes.fieldGroup}>
        {!isSmall && <Typography className={classes.label}>Energy</Typography>}
        {ENERGY_VALUES.map((value) => {
          const isSelected = value === currentEnergy
          const colorClass =
            value === 'low'
              ? classes.squareEnergyLow
              : value === 'medium'
                ? classes.squareEnergyMedium
                : classes.squareEnergyHigh
          const selectionClass = isSelected
            ? classes.selected
            : classes.unselected

          return (
            <div
              key={`energy-${value}`}
              className={`${classes.square} ${colorClass} ${selectionClass}`}
              onClick={(e) => handleEnergyClick(e, value)}
              title={`Energy: ${value.charAt(0).toUpperCase() + value.slice(1)}`}
            />
          )
        })}
      </div>

      {/* Mood */}
      <div className={classes.fieldGroup}>
        {!isSmall && <Typography className={classes.label}>Mood</Typography>}
        {MOOD_VALUES.map((value) => {
          const isSelected = value === currentMood
          const colorClass =
            value === 'negative'
              ? classes.squareMoodNegative
              : value === 'neutral'
                ? classes.squareMoodNeutral
                : classes.squareMoodPositive
          const selectionClass = isSelected
            ? classes.selected
            : classes.unselected

          return (
            <div
              key={`mood-${value}`}
              className={`${classes.square} ${colorClass} ${selectionClass}`}
              onClick={(e) => handleMoodClick(e, value)}
              title={`Mood: ${value.charAt(0).toUpperCase() + value.slice(1)}`}
            />
          )
        })}
      </div>
    </div>
  )
}

AlbumTagFields.propTypes = {
  size: PropTypes.oneOf(['small', 'medium']),
}

export default AlbumTagFields
