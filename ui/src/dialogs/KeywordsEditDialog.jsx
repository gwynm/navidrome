import React, { useState, useEffect, useCallback, useMemo } from 'react'
import PropTypes from 'prop-types'
import {
  useDataProvider,
  useNotify,
  useRefresh,
  useTranslate,
} from 'react-admin'
import {
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  TextField,
  Typography,
  makeStyles,
} from '@material-ui/core'
import Autocomplete from '@material-ui/lab/Autocomplete'
import { DialogTitle } from './DialogTitle'
import httpClient from '../dataProvider/httpClient'

const useStyles = makeStyles((theme) => ({
  section: {
    marginBottom: theme.spacing(2),
  },
  sectionLabel: {
    fontSize: '0.8rem',
    color: theme.palette.text.secondary,
    marginBottom: theme.spacing(0.5),
  },
  chipContainer: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.5),
  },
  currentChip: {
    margin: 0,
  },
  suggestionChip: {
    margin: 0,
    cursor: 'pointer',
  },
  emptyText: {
    fontSize: '0.8rem',
    color: theme.palette.text.disabled,
    fontStyle: 'italic',
  },
}))

const KeywordsEditDialog = ({
  open,
  onClose,
  trackIds,
  initialKeywords,
  resource,
}) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const refresh = useRefresh()
  const [keywords, setKeywords] = useState(initialKeywords || [])
  const [allKeywords, setAllKeywords] = useState([])
  const [inputValue, setInputValue] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setKeywords(initialKeywords || [])
  }, [initialKeywords])

  useEffect(() => {
    if (!open) return
    dataProvider
      .getList('tag', {
        filter: { tag_name: 'keyword' },
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'name', order: 'ASC' },
      })
      .then(({ data }) => {
        const values = data.map((t) => t.tagValue).filter(Boolean)
        setAllKeywords([...new Set(values)])
      })
      .catch(() => {})
  }, [open, dataProvider])

  const suggestions = useMemo(() => {
    const currentSet = new Set(keywords.map((k) => k.toLowerCase()))
    return allKeywords.filter((k) => !currentSet.has(k.toLowerCase()))
  }, [allKeywords, keywords])

  const autocompleteOptions = useMemo(() => {
    const currentSet = new Set(keywords.map((k) => k.toLowerCase()))
    return allKeywords.filter((k) => !currentSet.has(k.toLowerCase()))
  }, [allKeywords, keywords])

  const addKeyword = useCallback((value) => {
    const trimmed = value.trim()
    if (!trimmed) return
    setKeywords((prev) => {
      if (prev.some((k) => k.toLowerCase() === trimmed.toLowerCase()))
        return prev
      return [...prev, trimmed]
    })
  }, [])

  const removeKeyword = useCallback((value) => {
    setKeywords((prev) => prev.filter((k) => k !== value))
  }, [])

  const handleSave = useCallback(async () => {
    // Auto-commit any text left in the input field
    const finalKeywords = [...keywords]
    const pending = inputValue.trim()
    if (
      pending &&
      !finalKeywords.some((k) => k.toLowerCase() === pending.toLowerCase())
    ) {
      finalKeywords.push(pending)
    }

    setSaving(true)
    try {
      await Promise.all(
        trackIds.map((id) =>
          httpClient(`/api/song/${id}/keywords`, {
            method: 'PUT',
            body: JSON.stringify({ values: finalKeywords }),
          }),
        ),
      )
      refresh()
      onClose()
    } catch {
      notify('ra.page.error', 'warning')
    } finally {
      setSaving(false)
    }
  }, [trackIds, keywords, inputValue, onClose, notify, refresh])

  const handleKeyDown = useCallback(
    (e) => {
      if (e.key === 'Enter' && inputValue.trim()) {
        e.preventDefault()
        addKeyword(inputValue)
        setInputValue('')
      }
    },
    [inputValue, addKeyword],
  )

  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullWidth
      maxWidth="sm"
      onClick={(e) => e.stopPropagation()}
    >
      <DialogTitle onClose={onClose}>
        {translate('resources.song.actions.editKeywords', {
          _: 'Edit Keywords',
        })}
      </DialogTitle>
      <DialogContent>
        <div className={classes.section}>
          <Typography className={classes.sectionLabel}>
            {translate('resources.song.fields.keywords', {
              _: 'Current Keywords',
            })}
          </Typography>
          <div className={classes.chipContainer}>
            {keywords.length === 0 && (
              <Typography className={classes.emptyText}>
                {translate('resources.song.keywords.none', {
                  _: 'No keywords',
                })}
              </Typography>
            )}
            {keywords.map((kw) => (
              <Chip
                key={kw}
                label={kw}
                size="small"
                onDelete={() => removeKeyword(kw)}
                className={classes.currentChip}
              />
            ))}
          </div>
        </div>

        {suggestions.length > 0 && (
          <div className={classes.section}>
            <Typography className={classes.sectionLabel}>
              {translate('resources.song.keywords.suggestions', {
                _: 'Click to add',
              })}
            </Typography>
            <div className={classes.chipContainer}>
              {suggestions.map((kw) => (
                <Chip
                  key={kw}
                  label={kw}
                  size="small"
                  variant="outlined"
                  className={classes.suggestionChip}
                  onClick={() => addKeyword(kw)}
                />
              ))}
            </div>
          </div>
        )}

        <div className={classes.section}>
          <Autocomplete
            freeSolo
            value={null}
            options={autocompleteOptions}
            inputValue={inputValue}
            onInputChange={(_, val, reason) => {
              if (reason !== 'reset') setInputValue(val)
            }}
            onChange={(_, val) => {
              if (val) {
                addKeyword(val)
                setInputValue('')
              }
            }}
            renderInput={(params) => (
              <TextField
                {...params}
                label={translate('resources.song.keywords.addNew', {
                  _: 'Add keyword',
                })}
                variant="outlined"
                size="small"
                onKeyDown={handleKeyDown}
              />
            )}
          />
        </div>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="default">
          {translate('ra.action.cancel')}
        </Button>
        <Button onClick={handleSave} color="primary" disabled={saving}>
          {translate('ra.action.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

KeywordsEditDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  trackIds: PropTypes.arrayOf(PropTypes.string).isRequired,
  initialKeywords: PropTypes.arrayOf(PropTypes.string),
  resource: PropTypes.string,
}

export default KeywordsEditDialog
