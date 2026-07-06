import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import CommunityPage from './community/CommunityPage'
import config from './config'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
]

if (config.enableCommunity) {
  routes.push(
    <Route
      exact
      path="/community"
      render={() => <CommunityPage />}
      key={'community'}
    />,
  )
}

export default routes
