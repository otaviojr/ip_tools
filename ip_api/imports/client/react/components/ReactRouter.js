import React from 'react';
import {BrowserRouter, Route, Switch} from "react-router-dom";

import IpApp from './IpApp';
import NotFound from './NotFound';

const ReactRouter = () => (
  <BrowserRouter>
    <Switch>
      <Route exact path="/" component={IpApp}/>
      <Route component={NotFound}/>
    </Switch>
  </BrowserRouter>
);

export default ReactRouter;
