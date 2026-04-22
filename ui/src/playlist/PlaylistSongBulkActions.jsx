import React, { Fragment, useEffect, useMemo } from 'react'
import {
  BulkDeleteButton,
  useUnselectAll,
  useListContext,
  ResourceContextProvider,
} from 'react-admin'
import { MdOutlinePlaylistRemove } from 'react-icons/md'
import PropTypes from 'prop-types'
import { SongBulkActions } from '../common/SongBulkActions'

const PlaylistSongBulkActions = ({
  playlistId,
  readOnly,
  resource,
  selectedIds,
  onUnselectItems,
  ...rest
}) => {
  const { data } = useListContext()
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('playlistTrack')
  }, [unselectAll])

  // Map playlist track IDs to song (mediaFile) IDs for bulk actions
  const songIds = useMemo(
    () =>
      selectedIds
        .map((id) => data?.[id]?.mediaFileId || data?.[id]?.id)
        .filter(Boolean),
    [selectedIds, data],
  )

  const mappedResource = `playlist/${playlistId}/tracks`
  return (
    <Fragment>
      <SongBulkActions {...rest} selectedIds={songIds} resource="song" />
      {!readOnly && (
        <ResourceContextProvider value={mappedResource}>
          <BulkDeleteButton
            {...rest}
            selectedIds={selectedIds}
            label={'ra.action.remove'}
            icon={<MdOutlinePlaylistRemove />}
            resource={mappedResource}
            onClick={onUnselectItems}
          />
        </ResourceContextProvider>
      )}
    </Fragment>
  )
}

PlaylistSongBulkActions.propTypes = {
  playlistId: PropTypes.string.isRequired,
}

export default PlaylistSongBulkActions
