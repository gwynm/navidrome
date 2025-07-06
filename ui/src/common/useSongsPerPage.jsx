import { useSelector } from 'react-redux'
import { pageSizeMultiplier } from '../utils/pageSizes'

const getSongsPerPage = (width) => {
  // Use same logic as albums but with different base sizes for songs
  let baseSize
  if (width === 'xs') baseSize = 50
  else baseSize = 15
  return baseSize * pageSizeMultiplier()
}

const getSongsPerPageOptions = (width) => {
  // Use the same base options as albums but with different multipliers for songs
  const options = [15, 25, 50] // Base options for songs
  return options.map((size) => size * pageSizeMultiplier())
}

export const useSongsPerPage = (width) => {
  const perPage =
    useSelector(
      (state) => state?.admin.resources?.song?.list?.params?.perPage,
    ) || getSongsPerPage(width)

  return [perPage, getSongsPerPageOptions(width)]
}
