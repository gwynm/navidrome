import React from 'react'
import PotentialTrashList from './PotentialTrashList'
import DeleteSweepIcon from '@material-ui/icons/DeleteSweep'
import DeleteSweepOutlinedIcon from '@material-ui/icons/DeleteSweepOutlined'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'

export default {
  list: PotentialTrashList,
  icon: (
    <DynamicMenuIcon
      path={'potentialTrash'}
      icon={DeleteSweepOutlinedIcon}
      activeIcon={DeleteSweepIcon}
    />
  ),
}
