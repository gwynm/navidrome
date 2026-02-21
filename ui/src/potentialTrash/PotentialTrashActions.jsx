import React, { cloneElement } from 'react'
import { sanitizeListRestProps, TopToolbar } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { ToggleFieldsMenu } from '../common'
import DeleteTrashButton from './DeleteTrashButton'

export const PotentialTrashActions = ({
  className,
  resource,
  filters,
  displayedFilters,
  filterValues,
  showFilter,
  ...rest
}) => {
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {filters &&
        cloneElement(filters, {
          resource,
          showFilter,
          displayedFilters,
          filterValues,
          context: 'button',
        })}
      {isNotSmall && <ToggleFieldsMenu resource="potentialTrash" />}
      <DeleteTrashButton />
    </TopToolbar>
  )
}

PotentialTrashActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}
