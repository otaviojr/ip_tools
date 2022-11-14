import React from 'react';
import BaseApp from './BaseApp';
import { HTTP } from 'meteor/http';

import "../../../assets/stylesheets/app.css";
import "../../../assets/stylesheets/flags.css";

class IpApp extends BaseApp {
  state = {
    defaultIp: {},
    alternativeIp: {}
  };

  resolveIp = (force) => {
    let protocol = "ip";
    if(force) {
      protocol = force;
    }

    let port="";
    if(Meteor.isDevelopment){
      port=":3000";
    }

    api_url = `//${protocol}.otavio.guru${port}/api/v1/myip`;

    return new Promise((resolve,reject) => {
      HTTP.call("GET", api_url, {
          headers: {
          	"Accept": "application/json",
          }
      }, (error, result) => {
        if (error) {
          reject(error);
          return;
        }

        resolve(result);
      });
    });
  };

  componentDidMount(){
    this.resolveIp().then( (result) => {
      console.log("IP API result:");
      console.log(result);

      this.setState({
        defaultIp: result.data
      });

      const alternative = (result.data.ipv6 == true ? "ipv4" : "ipv6");
      this.resolveIp(alternative).then( (result) => {
        console.log("IP API alternate result:");
        console.log(result);

        this.setState({
          alternativeIp: result.data
        });

      }).catch( (error) => {
        console.log("Error calling IP API:");
        console.log(error);

        this.setState({
          alternativeIp: {
            ip: "No Alternate Ip Address",
            country_code: null
          }
        });

      });
    }).catch( (error) => {
      console.log("Error calling IP API:");
      console.log(error);

      this.setState({
        defaultIp: {
          ip: "ERROR",
          country_code: null
        },
        alternativeIp: {
          ip: "ERROR",
          country_code: null
        }
      });

    });
  }

  render() {

    let defaultFlag = null;
    let alternativeFlag = null;

    if(this.state.defaultIp && this.state.defaultIp.country_code){
      defaultFlag = <span className={`flag-icon flag-icon-${this.state.defaultIp.country_code.toLowerCase()}`}></span>
    }

    if(this.state.alternativeIp && this.state.alternativeIp.country_code){
      alternativeFlag = <span className={`flag-icon flag-icon-${this.state.alternativeIp.country_code.toLowerCase()}`}></span>
    }

    return (
      <div className="ip-container">
        <p className="default">{defaultFlag}{this.state.defaultIp.ip}</p>
        <p className="alternative">{alternativeFlag}{this.state.alternativeIp.ip}</p>
      </div>
    );
  }
}

export default IpApp;
