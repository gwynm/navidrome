import React, { useState } from 'react'
import DeleteIcon from '@material-ui/icons/Delete'
import { makeStyles, alpha } from '@material-ui/core/styles'
import clsx from 'clsx'
import {
  Button,
  Confirm,
  useNotify,
  useRefresh,
  useUnselectAll,
} from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const useStyles = makeStyles(
  (theme) => ({
    deleteButton: {
      color: theme.palette.error.main,
      '&:hover': {
        backgroundColor: alpha(theme.palette.error.main, 0.12),
        '@media (hover: none)': {
          backgroundColor: 'transparent',
        },
      },
    },
  }),
  { name: 'DeleteTrashButton' },
)

const DeleteTrashButton = ({ className }) => {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const unselectAll = useUnselectAll()
  const refresh = useRefresh()
  const notify = useNotify()
  const classes = useStyles()

  const handleClick = () => setOpen(true)
  const handleDialogClose = () => setOpen(false)
  const handleConfirm = () => {
    setLoading(true)
    setOpen(false)
    httpClient(`${REST_URL}/potentialTrash`, { method: 'DELETE' })
      .then(() => {
        notify('resources.potentialTrash.notifications.removed')
        unselectAll('potentialTrash')
        refresh()
      })
      .catch(() => {
        notify('resources.potentialTrash.notifications.error', {
          type: 'warning',
        })
      })
      .finally(() => setLoading(false))
  }

  return (
    <>
      <Button
        onClick={handleClick}
        label={'resources.potentialTrash.actions.deleteAll'}
        key="button"
        disabled={loading}
        className={clsx('ra-delete-button', classes.deleteButton, className)}
      >
        <DeleteIcon />
      </Button>
      <Confirm
        isOpen={open}
        loading={loading}
        title={'message.delete_trash_title'}
        content={'message.delete_trash_content'}
        onConfirm={handleConfirm}
        onClose={handleDialogClose}
      />
    </>
  )
}

export default DeleteTrashButton
