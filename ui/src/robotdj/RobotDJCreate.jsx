import React, { useState, useCallback } from 'react'
import {
  Card,
  CardContent,
  TextField,
  Button,
  Typography,
  RadioGroup,
  FormControlLabel,
  Radio,
  FormControl,
  FormLabel,
  CircularProgress,
  List,
  ListItem,
  ListItemText,
  Divider,
  Box,
  Paper,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { useRedirect, useNotify, useTranslate } from 'react-admin'
import { Title } from '../common'
import { baseUrl } from '../utils'

const useStyles = makeStyles((theme) => ({
  root: {
    maxWidth: 800,
    margin: '0 auto',
    padding: theme.spacing(2),
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(2),
    marginBottom: theme.spacing(3),
  },
  buttonRow: {
    display: 'flex',
    gap: theme.spacing(2),
    alignItems: 'center',
    marginTop: theme.spacing(1),
  },
  previewSection: {
    marginTop: theme.spacing(3),
  },
  rulesBox: {
    padding: theme.spacing(2),
    marginBottom: theme.spacing(2),
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    maxHeight: 200,
    overflow: 'auto',
    backgroundColor:
      theme.palette.type === 'dark'
        ? theme.palette.background.default
        : theme.palette.grey[100],
  },
  trackList: {
    maxHeight: 400,
    overflow: 'auto',
  },
  trackCount: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}))

const RobotDJCreate = () => {
  const classes = useStyles()
  const redirect = useRedirect()
  const notify = useNotify()
  const translate = useTranslate()

  const [name, setName] = useState('')
  const [criteria, setCriteria] = useState('')
  const [mode, setMode] = useState('strict')
  const [loading, setLoading] = useState(false)
  const [preview, setPreview] = useState(null)
  const [confirmDialog, setConfirmDialog] = useState(null)

  const getAuthHeaders = useCallback(() => {
    const headers = { 'Content-Type': 'application/json' }
    const token = localStorage.getItem('token')
    if (token) {
      headers['X-ND-Authorization'] = `Bearer ${token}`
    }
    return headers
  }, [])

  const handlePreview = useCallback(async () => {
    if (!criteria.trim()) {
      notify('Please enter criteria for your playlist', 'warning')
      return
    }

    setLoading(true)
    setPreview(null)

    try {
      const response = await fetch(baseUrl('/api/ai/playlist/preview'), {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          name: name || 'Robot DJ Playlist',
          criteria,
          mode,
          confirmed: false,
        }),
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text)
      }

      const result = await response.json()

      if (result.needsConfirm) {
        setConfirmDialog(result)
      } else {
        setPreview(result)
      }
    } catch (err) {
      notify(`Preview failed: ${err.message}`, 'error')
    } finally {
      setLoading(false)
    }
  }, [criteria, mode, name, notify, getAuthHeaders])

  const handleVibesConfirm = useCallback(async () => {
    setConfirmDialog(null)
    setLoading(true)

    try {
      const response = await fetch(baseUrl('/api/ai/playlist/preview'), {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          name: name || 'Robot DJ Playlist',
          criteria,
          mode: 'vibes',
          confirmed: true,
        }),
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text)
      }

      const result = await response.json()
      setPreview(result)
    } catch (err) {
      notify(`Preview failed: ${err.message}`, 'error')
    } finally {
      setLoading(false)
    }
  }, [criteria, name, notify, getAuthHeaders])

  const handleConfirmPlaylist = useCallback(async () => {
    if (!preview || !preview.tracks.length) return

    setLoading(true)
    try {
      const response = await fetch(baseUrl('/api/ai/playlist/confirm'), {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          name: name || 'Robot DJ Playlist',
          trackIds: preview.tracks.map((t) => t.id),
          rules: preview.rules,
          criteria,
        }),
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text)
      }

      const result = await response.json()
      notify('Playlist created!', 'success')
      redirect(`/playlist/${result.playlistId}/show`)
    } catch (err) {
      notify(`Failed to create playlist: ${err.message}`, 'error')
    } finally {
      setLoading(false)
    }
  }, [preview, name, criteria, notify, redirect, getAuthHeaders])

  return (
    <div className={classes.root}>
      <Title subTitle="Robot DJ" />
      <Card>
        <CardContent>
          <Typography variant="h5" gutterBottom>
            Robot DJ
          </Typography>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            Describe the playlist you want, and AI will create it for you.
          </Typography>

          <div className={classes.form}>
            <TextField
              label="Playlist Name"
              variant="outlined"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My AI Playlist"
              fullWidth
            />
            <TextField
              label="Criteria"
              variant="outlined"
              multiline
              minRows={3}
              value={criteria}
              onChange={(e) => setCriteria(e.target.value)}
              placeholder='e.g. "Upbeat rock songs from the 90s" or "Chill jazz for late night studying"'
              fullWidth
            />
            <FormControl component="fieldset">
              <FormLabel component="legend">Mode</FormLabel>
              <RadioGroup
                row
                value={mode}
                onChange={(e) => setMode(e.target.value)}
              >
                <FormControlLabel
                  value="strict"
                  control={<Radio />}
                  label="Strict (filter-based)"
                />
                <FormControlLabel
                  value="vibes"
                  control={<Radio />}
                  label="Vibes (AI picks songs directly)"
                />
              </RadioGroup>
            </FormControl>

            <div className={classes.buttonRow}>
              <Button
                variant="contained"
                color="primary"
                onClick={handlePreview}
                disabled={loading || !criteria.trim()}
              >
                {loading ? <CircularProgress size={24} /> : 'Preview'}
              </Button>
            </div>
          </div>

          {preview && (
            <div className={classes.previewSection}>
              <Divider />
              <Typography variant="h6" gutterBottom style={{ marginTop: 16 }}>
                Preview
              </Typography>

              <Typography variant="subtitle2">Rules</Typography>
              <Paper className={classes.rulesBox} variant="outlined">
                {preview.rules}
              </Paper>

              <Typography variant="subtitle2" className={classes.trackCount}>
                {preview.tracks.length} track
                {preview.tracks.length !== 1 ? 's' : ''} found
              </Typography>

              <Paper variant="outlined" className={classes.trackList}>
                <List dense>
                  {preview.tracks.map((track, idx) => (
                    <ListItem key={track.id || idx}>
                      <ListItemText
                        primary={`${track.artist} — ${track.title}`}
                      />
                    </ListItem>
                  ))}
                </List>
              </Paper>

              <Box mt={2}>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={handleConfirmPlaylist}
                  disabled={loading || preview.tracks.length === 0}
                >
                  {loading ? (
                    <CircularProgress size={24} />
                  ) : (
                    'Create Playlist'
                  )}
                </Button>
              </Box>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={!!confirmDialog}
        onClose={() => setConfirmDialog(null)}
      >
        <DialogTitle>Vibes Mode - Large Data Warning</DialogTitle>
        <DialogContent>
          <DialogContentText>
            This will send your entire music library metadata to the AI for
            analysis.
            <br />
            <br />
            <strong>Library size:</strong> {confirmDialog?.csvRows} tracks
            <br />
            <strong>Data size:</strong>{' '}
            {confirmDialog?.csvSizeBytes
              ? `${(confirmDialog.csvSizeBytes / 1024 / 1024).toFixed(1)} MB`
              : 'unknown'}
            <br />
            <strong>Estimated tokens:</strong>{' '}
            {confirmDialog?.estimatedTokens?.toLocaleString()}
            <br />
            <br />
            This API call may be expensive. Do you want to proceed?
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDialog(null)} color="default">
            Cancel
          </Button>
          <Button onClick={handleVibesConfirm} color="primary">
            Proceed
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  )
}

export default RobotDJCreate
