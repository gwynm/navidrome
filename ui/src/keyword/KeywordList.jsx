import React, { useState, useCallback } from 'react'
import {
  Datagrid,
  Filter,
  NumberField,
  SearchInput,
  TextField,
  useTranslate,
  useNotify,
  useRefresh,
} from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  TextField as MuiTextField,
  makeStyles,
} from '@material-ui/core'
import EditIcon from '@material-ui/icons/Edit'
import DeleteIcon from '@material-ui/icons/Delete'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import { useHistory } from 'react-router-dom'
import { List } from '../common'
import httpClient from '../dataProvider/httpClient'

const useStyles = makeStyles({
  actions: {
    whiteSpace: 'nowrap',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
  },
})

const KeywordFilter = (props) => (
  <Filter {...props} variant="outlined">
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const KeywordActions = ({ record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const history = useHistory()
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [saving, setSaving] = useState(false)

  const handleEditOpen = useCallback(
    (e) => {
      e.stopPropagation()
      setNewName(record.tagValue)
      setEditOpen(true)
    },
    [record],
  )

  const handleEditSave = useCallback(async () => {
    const trimmed = newName.trim()
    if (!trimmed) return
    setSaving(true)
    try {
      await httpClient(`/api/keyword/${record.id}`, {
        method: 'PUT',
        body: JSON.stringify({ name: trimmed }),
      })
      notify('resources.keyword.notifications.renamed', 'info')
      refresh()
    } catch {
      notify('resources.keyword.notifications.error', 'warning')
    } finally {
      setSaving(false)
      setEditOpen(false)
    }
  }, [record, newName, notify, refresh])

  const handleDelete = useCallback(async () => {
    setSaving(true)
    try {
      await httpClient(`/api/keyword/${record.id}`, {
        method: 'DELETE',
      })
      notify('resources.keyword.notifications.deleted', 'info')
      refresh()
    } catch {
      notify('resources.keyword.notifications.error', 'warning')
    } finally {
      setSaving(false)
      setDeleteOpen(false)
    }
  }, [record, notify, refresh])

  const handleTracks = useCallback(
    (e) => {
      e.stopPropagation()
      history.push(
        `/song?filter=${encodeURIComponent(JSON.stringify({ keyword: [record.id] }))}`,
      )
    },
    [record, history],
  )

  if (!record) return null

  return (
    <div className={classes.actions}>
      <IconButton
        size="small"
        onClick={handleEditOpen}
        title={translate('ra.action.edit')}
      >
        <EditIcon fontSize="small" />
      </IconButton>
      <IconButton
        size="small"
        onClick={(e) => {
          e.stopPropagation()
          setDeleteOpen(true)
        }}
        title={translate('ra.action.delete')}
      >
        <DeleteIcon fontSize="small" />
      </IconButton>
      <IconButton
        size="small"
        onClick={handleTracks}
        title={translate('resources.keyword.actions.tracks')}
      >
        <QueueMusicIcon fontSize="small" />
      </IconButton>

      <Dialog
        open={editOpen}
        onClose={() => setEditOpen(false)}
        onClick={(e) => e.stopPropagation()}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle>
          {translate('resources.keyword.actions.rename')}
        </DialogTitle>
        <DialogContent>
          <MuiTextField
            autoFocus
            fullWidth
            variant="outlined"
            size="small"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleEditSave()
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>
            {translate('ra.action.cancel')}
          </Button>
          <Button
            onClick={handleEditSave}
            color="primary"
            disabled={saving || !newName.trim()}
          >
            {translate('ra.action.save')}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onClick={(e) => e.stopPropagation()}
      >
        <DialogTitle>
          {translate('resources.keyword.actions.confirmDelete')}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {translate('resources.keyword.actions.confirmDeleteMessage', {
              keyword: record.tagValue,
            })}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteOpen(false)}>
            {translate('ra.action.cancel')}
          </Button>
          <Button onClick={handleDelete} color="secondary" disabled={saving}>
            {translate('ra.action.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  )
}

const KeywordList = (props) => {
  return (
    <List
      {...props}
      exporter={false}
      sort={{ field: 'name', order: 'ASC' }}
      bulkActionButtons={false}
      filters={<KeywordFilter />}
      perPage={25}
    >
      <Datagrid>
        <TextField source="tagValue" label="resources.keyword.fields.name" />
        <NumberField
          source="songCount"
          label="resources.keyword.fields.trackCount"
        />
        <KeywordActions source="id" label="" sortable={false} />
      </Datagrid>
    </List>
  )
}

export default KeywordList
