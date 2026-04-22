import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import RobotDJCreate from './robotdj/RobotDJCreate'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  <Route
    exact
    path="/robotdj"
    render={() => <RobotDJCreate />}
    key={'robotdj'}
  />,
]

export default routes
