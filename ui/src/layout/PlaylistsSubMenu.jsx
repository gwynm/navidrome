import React, { useCallback, useState } from 'react'
import {
  MenuItemLink,
  useDataProvider,
  useNotify,
  useQueryWithStore,
} from 'react-admin'
import { useHistory } from 'react-router-dom'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import { Typography } from '@material-ui/core'
import QueueMusicOutlinedIcon from '@material-ui/icons/QueueMusicOutlined'
import { BiCog } from 'react-icons/bi'
import { useDrop } from 'react-dnd'
import SubMenu from './SubMenu'
import { canChangeTracks } from '../common'
import { DraggableTypes } from '../consts'
import config from '../config'
import ExpandMore from '@material-ui/icons/ExpandMore'
import ArrowRightOutlined from '@material-ui/icons/ArrowRightOutlined'
import Collapse from '@material-ui/core/Collapse'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles((theme) => ({
  folderHeader: {
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    '& .MuiTypography-root': {
      marginLeft: theme.spacing(1),
    },
  },
}))

const PlaylistMenuItemLink = ({ pls, sidebarIsOpen }) => {
  const dataProvider = useDataProvider()
  const notify = useNotify()

  const [, dropRef] = useDrop(() => ({
    accept: canChangeTracks(pls) ? DraggableTypes.ALL : [],
    drop: (item) =>
      dataProvider
        .addToPlaylist(pls.id, item)
        .then((res) => {
          notify('message.songsAddedToPlaylist', 'info', {
            smart_count: res.data?.added,
          })
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        }),
  }))

  return (
    <MenuItemLink
      to={`/playlist/${pls.id}/show`}
      primaryText={
        <Typography variant="inherit" noWrap ref={dropRef}>
          {pls.name}
        </Typography>
      }
      sidebarIsOpen={sidebarIsOpen}
      dense={false}
    />
  )
}

const parseFolderAndName = (fullName) => {
  const parts = fullName.split('>')
  if (parts.length === 1) return { folder: 'Unsorted', name: parts[0].trim() }
  return { folder: parts[0].trim(), name: parts[1].trim() }
}

const PlaylistsSubMenu = ({ state, setState, sidebarIsOpen, dense }) => {
  const history = useHistory()
  const { data, loaded } = useQueryWithStore({
    type: 'getList',
    resource: 'playlist',
    payload: {
      pagination: {
        page: 0,
        perPage: config.maxSidebarPlaylists,
      },
      sort: { field: 'name' },
    },
  })

  const handleToggle = (menu) => {
    setState((state) => ({ ...state, [menu]: !state[menu] }))
  }

  const renderPlaylistMenuItemLink = (pls) => (
    <PlaylistMenuItemLink
      pls={pls}
      sidebarIsOpen={sidebarIsOpen}
      key={pls.id}
    />
  )

  const groupPlaylistsByFolder = (playlists) => {
    const groups = new Map()
    playlists.forEach((pls) => {
      const { folder, name } = parseFolderAndName(pls.name)
      if (!groups.has(folder)) {
        groups.set(folder, [])
      }
      groups.get(folder).push({ ...pls, displayName: name })
    })
    return groups
  }

  const [openFolders, setOpenFolders] = useState({})
  const classes = useStyles()

  const toggleFolder = (folder) => {
    setOpenFolders((prev) => ({
      ...prev,
      [folder]: !prev[folder],
    }))
  }

  const renderPlaylistGroup = (playlists, folder) => {
    if (!folder) {
      return playlists.map((pls) => (
        <PlaylistMenuItemLink
          pls={{ ...pls, name: pls.displayName }}
          sidebarIsOpen={sidebarIsOpen}
          key={pls.id}
        />
      ))
    }

    return (
      <React.Fragment key={folder}>
        <PlaylistFolderHeader
          title={folder}
          isOpen={openFolders[folder]}
          onClick={() => toggleFolder(folder)}
        />
        <Collapse in={openFolders[folder]} timeout="auto">
          {playlists.map((pls) => (
            <PlaylistMenuItemLink
              pls={{ ...pls, name: pls.displayName }}
              sidebarIsOpen={sidebarIsOpen}
              key={pls.id}
            />
          ))}
        </Collapse>
      </React.Fragment>
    )
  }

  const userId = localStorage.getItem('userId')
  const myPlaylists = []
  const sharedPlaylists = []

  if (loaded && data) {
    const allPlaylists = Object.keys(data).map((id) => data[id])

    allPlaylists.forEach((pls) => {
      if (userId === pls.ownerId) {
        myPlaylists.push(pls)
      } else {
        sharedPlaylists.push(pls)
      }
    })
  }

  const groupedMyPlaylists = groupPlaylistsByFolder(myPlaylists)
  const groupedSharedPlaylists = groupPlaylistsByFolder(sharedPlaylists)

  const onPlaylistConfig = useCallback(
    () => history.push('/playlist'),
    [history],
  )

  return (
    <>
      <SubMenu
        handleToggle={() => handleToggle('menuPlaylists')}
        isOpen={state.menuPlaylists}
        sidebarIsOpen={sidebarIsOpen}
        name={'menu.playlists'}
        icon={<QueueMusicIcon />}
        dense={dense}
        actionIcon={<BiCog />}
        onAction={onPlaylistConfig}
      >
        {Array.from(groupedMyPlaylists.entries()).map(([folder, playlists]) =>
          renderPlaylistGroup(playlists, folder),
        )}
      </SubMenu>
      {sharedPlaylists?.length > 0 && (
        <SubMenu
          handleToggle={() => handleToggle('menuSharedPlaylists')}
          isOpen={state.menuSharedPlaylists}
          sidebarIsOpen={sidebarIsOpen}
          name={'menu.sharedPlaylists'}
          icon={<QueueMusicOutlinedIcon />}
          dense={dense}
        >
          {Array.from(groupedSharedPlaylists.entries()).map(
            ([folder, playlists]) => renderPlaylistGroup(playlists, folder),
          )}
        </SubMenu>
      )}
    </>
  )
}

const PlaylistFolderHeader = ({ title, isOpen, onClick }) => {
  const classes = useStyles()
  return (
    <div className={classes.folderHeader} onClick={onClick}>
      {isOpen ? (
        <ExpandMore fontSize="small" color="primary" />
      ) : (
        <ArrowRightOutlined fontSize="small" color="primary" />
      )}
      <Typography color="primary" noWrap>
        {title}
      </Typography>
    </div>
  )
}

export default PlaylistsSubMenu
