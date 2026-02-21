import { useState, useEffect } from 'react'
import { useNotify, useRefresh } from 'react-admin'
import subsonic from '../subsonic'

export const useRating = (resource, record, afterRate) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const refresh = useRefresh()
  const [localRating, setLocalRating] = useState(record.rating)

  useEffect(() => {
    setLocalRating(record.rating)
  }, [record.rating])

  const rate = (val, id) => {
    setLocalRating(val)
    setLoading(true)
    subsonic
      .setRating(id, val)
      .then(async () => {
        if (afterRate) {
          try {
            await afterRate(val)
          } catch (e) {
            // eslint-disable-next-line no-console
            console.log('Error in afterRate callback: ', e)
          }
        }
      })
      .then(() => {
        if (afterRate) {
          setTimeout(() => {
            refresh()
            setLoading(false)
          }, 1000)
        } else {
          refresh()
          setLoading(false)
        }
      })
      .catch((e) => {
        setLocalRating(record.rating)
        // eslint-disable-next-line no-console
        console.log('Error setting star rating: ', e)
        notify('ra.page.error', 'warning')
        setLoading(false)
      })
  }

  return [rate, localRating, loading]
}
