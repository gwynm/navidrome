import React, { useState, useCallback, useMemo, useRef } from 'react'
import PropTypes from 'prop-types'
import { useDataProvider, useNotify, useRecordContext } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import httpClient from '../dataProvider/httpClient'

const ENERGY_VALUES = ['low', 'medium', 'high']
const EMPTY_RECORD = {}

// Cache for suggested energy results to avoid repeated API calls
const suggestedEnergyCache = new Map()

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
  // Suggestion dot indicator
  suggestionDot: {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    width: '4px',
    height: '4px',
    borderRadius: '50%',
    backgroundColor: theme.palette.common.white,
    boxShadow: '0 0 2px rgba(0,0,0,0.5)',
  },
  squareWithSuggestion: {
    position: 'relative',
  },
  analyzing: {
    animation: '$pulse 1s infinite',
  },
  '@keyframes pulse': {
    '0%, 100%': { opacity: 0.5 },
    '50%': { opacity: 1 },
  },
}))

// Build tooltip text showing score breakdown
const buildScoreTooltip = (value, analysisResult) => {
  const label = value.charAt(0).toUpperCase() + value.slice(1)
  if (!analysisResult) return label

  const { suggestedEnergy, metrics, score } = analysisResult
  if (value !== suggestedEnergy) return label

  // Handle case where score or metrics might not be present
  if (!score || !metrics) {
    return `${label} (suggested)`
  }

  const lines = [
    `${label} (suggested)`,
    `Score: ${score.totalScore} (≥5=high, ≥2=medium, <2=low)`,
    `  BPM ${Math.round(metrics.bpm || 0)}: +${score.bpmScore}`,
    `  Beats ${(metrics.beatsLoudness || 0).toFixed(2)}: +${score.beatsScore}`,
    `  Loudness ${(metrics.averageLoudness || 0).toFixed(2)}: +${score.loudnessScore}`,
    `  Dance ${(metrics.danceability || 0).toFixed(2)}: +${score.danceabilityScore}`,
  ]
  return lines.join('\n')
}

export const EnergyField = ({ resource, ...rest }) => {
  const contextRecord = useRecordContext(rest)
  const record = useMemo(() => contextRecord || EMPTY_RECORD, [contextRecord])
  const classes = useStyles()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [loading, setLoading] = useState(false)
  const [analysisResult, setAnalysisResult] = useState(null)
  const [analyzing, setAnalyzing] = useState(false)
  const fetchedRef = useRef(false)

  // Get current energy value from tags
  const currentEnergy = record?.tags?.energy?.[0] || ''
  const suggestedEnergy = analysisResult?.suggestedEnergy

  // Fetch suggested energy on first hover
  const fetchSuggestedEnergy = useCallback(async () => {
    const id = record.mediaFileId || record.id
    if (!id || fetchedRef.current || analyzing) return

    // Check cache first
    if (suggestedEnergyCache.has(id)) {
      setAnalysisResult(suggestedEnergyCache.get(id))
      fetchedRef.current = true
      return
    }

    fetchedRef.current = true
    setAnalyzing(true)

    try {
      const response = await httpClient(`/api/song/${id}/suggested-energy`)
      const data = await response.json
      if (data?.suggestedEnergy) {
        suggestedEnergyCache.set(id, data)
        setAnalysisResult(data)
      }
    } catch {
      // Silently fail - analyzer may not be available
      fetchedRef.current = false // Allow retry on next hover
    } finally {
      setAnalyzing(false)
    }
  }, [record, analyzing])

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
      const newValue = value === currentEnergy ? '' : value
      setEnergy(newValue)
    },
    [currentEnergy, setEnergy, record?.missing, loading],
  )

  const isDisabled = record?.missing
  const containerClass = `${classes.container} ${isDisabled ? classes.disabled : ''} ${loading ? classes.loading : ''}`

  return (
    <div
      className={containerClass}
      onClick={(e) => e.stopPropagation()}
      onMouseEnter={fetchSuggestedEnergy}
    >
      {ENERGY_VALUES.map((value) => {
        const isSelected = value === currentEnergy
        const isSuggested = value === suggestedEnergy
        const squareColorClass =
          value === 'low'
            ? classes.squareLow
            : value === 'medium'
              ? classes.squareMedium
              : classes.squareHigh
        const selectionClass = isSelected
          ? classes.selected
          : classes.unselected
        const suggestionClass = isSuggested ? classes.squareWithSuggestion : ''
        const analyzingClass = analyzing ? classes.analyzing : ''

        const title = buildScoreTooltip(value, analysisResult)

        return (
          <div
            key={value}
            className={`${classes.square} ${squareColorClass} ${selectionClass} ${suggestionClass} ${analyzingClass}`}
            onClick={(e) => handleSquareClick(e, value)}
            title={title}
          >
            {isSuggested && <div className={classes.suggestionDot} />}
          </div>
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
