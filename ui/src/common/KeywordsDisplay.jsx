import React, { useState, useCallback, useEffect, useMemo } from 'react'
import PropTypes from 'prop-types'
import { useDataProvider, useTranslate, useVersion } from 'react-admin'
import { Chip, makeStyles } from '@material-ui/core'
import KeywordsEditDialog from '../dialogs/KeywordsEditDialog'

const useStyles = makeStyles((theme) => ({
  container: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.5),
    marginTop: theme.spacing(0.5),
    cursor: 'pointer',
  },
  chip: {
    height: 20,
    fontSize: '0.72rem',
    '& .MuiChip-label': {
      padding: '0 6px',
    },
  },
}))

export const KeywordsDisplay = ({ albumId, playlistId }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const version = useVersion()
  const [songs, setSongs] = useState([])
  const [dialogOpen, setDialogOpen] = useState(false)

  useEffect(() => {
    const filter = albumId ? { album_id: albumId } : { playlist_id: playlistId }
    const resource = albumId ? 'song' : 'playlistTrack'

    dataProvider
      .getList(resource, {
        filter,
        pagination: { page: 1, perPage: 1000 },
        sort: { field: 'album', order: 'ASC' },
      })
      .then(({ data }) => {
        setSongs(data || [])
      })
      .catch(() => {})
  }, [albumId, playlistId, dataProvider, version])

  const allKeywords = useMemo(() => {
    const set = new Set()
    songs.forEach((song) => {
      const kws = song?.tags?.keyword || []
      kws.forEach((k) => set.add(k))
    })
    return [...set].sort()
  }, [songs])

  const trackIds = useMemo(
    () => songs.map((s) => s.mediaFileId || s.id),
    [songs],
  )

  const handleClick = useCallback((e) => {
    e.stopPropagation()
    setDialogOpen(true)
  }, [])

  const handleClose = useCallback(() => {
    setDialogOpen(false)
  }, [])

  if (allKeywords.length === 0) return null

  return (
    <>
      <div
        className={classes.container}
        onClick={handleClick}
        title={translate('resources.song.actions.editKeywords', {
          _: 'Edit Keywords',
        })}
      >
        {allKeywords.map((kw) => (
          <Chip
            key={kw}
            label={kw}
            size="small"
            variant="outlined"
            className={classes.chip}
          />
        ))}
      </div>
      {dialogOpen && (
        <KeywordsEditDialog
          open={dialogOpen}
          onClose={handleClose}
          trackIds={trackIds}
          initialKeywords={allKeywords}
          resource="song"
        />
      )}
    </>
  )
}

KeywordsDisplay.propTypes = {
  albumId: PropTypes.string,
  playlistId: PropTypes.string,
}

export default KeywordsDisplay
