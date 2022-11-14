import React from 'react';
import { Meteor } from 'meteor/meteor';
import mysql from 'mysql';

import {DatabaseConfig} from '../imports/server/config/Config';

import Api from '../imports/server/api/Api';

Meteor.startup(() => {
  let config = null;

  if(Meteor.isDevelopment){
    config = DatabaseConfig["development"];
  } else {
    config = DatabaseConfig["production"]
  }

  var connection = mysql.createConnection(config);

  connection.connect(function(err) {
    if (err) {
      console.error('error connecting: ' + err.stack);
      return;
    }

    console.log('connected as id ' + connection.threadId);
  });

  /* Configuring Router with out API's Endpoints */
  Router.route('/api/v1/:request', function () {
    console.log(this.params.request + ":" + this.request.method);
    this.response.setHeader( 'Access-Control-Allow-Origin', '*' );
    this.response.setHeader( 'Content-Type', 'application/json' );
    if ( this.request.method === "OPTIONS" ) {
      this.response.setHeader( 'Access-Control-Allow-Headers', 'Origin, X-Requested-With, Content-Type, Accept' );
      this.response.setHeader( 'Access-Control-Allow-Methods', 'POST, PUT, GET, DELETE, OPTIONS' );
      this.response.end( 'Set OPTIONS.' );
    } else {
      Api.handleRequest(this.params.request, this.request, this.response, this.params, {
        mysqlConn: connection
      });
    }
  }, {where: 'server'});
});
