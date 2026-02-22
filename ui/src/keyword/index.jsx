import React from 'react'
import KeywordList from './KeywordList'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import LabelIcon from '@material-ui/icons/Label'
import LabelOutlinedIcon from '@material-ui/icons/LabelOutlined'

const keyword = {
  list: KeywordList,
  icon: (
    <DynamicMenuIcon
      path="keyword"
      icon={LabelOutlinedIcon}
      activeIcon={LabelIcon}
    />
  ),
}

export default keyword
