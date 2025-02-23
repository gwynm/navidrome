import { useState, useCallback, useEffect, useRef } from 'react'
import { useDataProvider, useNotify, useRefresh } from 'react-admin'
import subsonic from '../subsonic'

export const useRating = (resource, record) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const refresh = useRefresh()
  const dataProvider = useDataProvider()
  const mountedRef = useRef(false)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refreshRating = useCallback(() => {
    const actualResource = resource === 'playlistTrack' ? 'song' : resource
    const id = resource === 'playlistTrack' ? record.mediaFileId : record.id
    dataProvider
      .getOne(actualResource, { id })
      .then(() => {
        if (mountedRef.current) {
          setLoading(false)
          if (resource === 'playlistTrack') refresh() // brute force recovery by refreshing all. Would be better to figure out why refreshing the song doesn't seem to do it
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error encountered: ' + e)
      })
  }, [dataProvider, record, resource])

  const rate = (val, id) => {
    setLoading(true)
    subsonic
      .setRating(id, val)
      .then(refreshRating)
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error setting star rating: ', e)
        notify('ra.page.error', 'warning')
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }

  return [rate, record.rating, loading]
}
