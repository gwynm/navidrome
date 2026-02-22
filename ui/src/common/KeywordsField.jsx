import React, { useMemo, useState, useCallback } from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'
import { Chip, makeStyles } from '@material-ui/core'
import KeywordsEditDialog from '../dialogs/KeywordsEditDialog'

const EMPTY_RECORD = {}

const useStyles = makeStyles((theme) => ({
  container: {
    display: 'inline-flex',
    flexWrap: 'wrap',
    gap: '2px',
    padding: '2px 0',
    cursor: 'pointer',
    minWidth: 40,
    minHeight: 20,
  },
  chip: {
    height: 18,
    fontSize: '0.7rem',
    '& .MuiChip-label': {
      padding: '0 6px',
    },
  },
  disabled: {
    opacity: 0.3,
    pointerEvents: 'none',
  },
}))

export const KeywordsField = ({ resource, ...rest }) => {
  const contextRecord = useRecordContext(rest)
  const record = useMemo(() => contextRecord || EMPTY_RECORD, [contextRecord])
  const classes = useStyles()
  const [dialogOpen, setDialogOpen] = useState(false)

  const keywords = record?.tags?.keyword || []
  const trackId = record.mediaFileId || record.id

  const handleClick = useCallback(
    (e) => {
      e.stopPropagation()
      if (record?.missing) return
      setDialogOpen(true)
    },
    [record?.missing],
  )

  const handleClose = useCallback(() => {
    setDialogOpen(false)
  }, [])

  const isDisabled = record?.missing

  return (
    <>
      <div
        className={`${classes.container} ${isDisabled ? classes.disabled : ''}`}
        onClick={handleClick}
      >
        {keywords.map((kw) => (
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
          trackIds={[trackId]}
          initialKeywords={keywords}
          resource={resource}
        />
      )}
    </>
  )
}

KeywordsField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object,
}

export default KeywordsField
