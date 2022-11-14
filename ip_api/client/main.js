import { Meteor } from 'meteor/meteor';
import React from 'react';
import { render } from 'react-dom';

Router.options.autoStart = false;

import ReactRouter from '../imports/client/react/components/ReactRouter';

render(<ReactRouter />, document.querySelector('#main'));
