class Api {
  static handleRequest = (method, request, response, params, options) => {
    switch(method){
      case "myip":
        Api.handleMyIp(request, response, params, options);
      default:
        Api.handleNotFound(request, response, params, options);
    }
  };

  static handleMyIp = (request, response,  params,  options) => {
    let clientIp = request.connection.remoteAddress;

    if(request.headers['x-forwarded-for']){
      clientIp = request.headers['x-forwarded-for'].split(",")[0];
    }

    //force for development purpose
    if(Meteor.isDevelopment){
      clientIp = "2804:431:9705:2074:65d3:32b6:4bed:3e22";
    }

    let isIPV6 = (clientIp.indexOf(":") >= 0);

    let query=null;
    if(isIPV6){
      query="SELECT country_code from iana_records where reg_type=? and ip_start_range <= INET6_ATON(?) and ip_end_range >= INET6_ATON(?)";
    } else {
      query="SELECT country_code from iana_records where reg_type=? and ip_start_range <= INET_ATON(?) and ip_end_range >= INET_ATON(?)";
    }

    options.mysqlConn.query(query,[(isIPV6 ? "ipv6" : "ipv4"), clientIp, clientIp], function(error, results, fields){
      let country_code = "unknown";
      if(!error){
        if(results && results.length > 0){
          country_code = results[0].country_code;
        }
      }

      if(!params.query.field){
        const json = {
          ret: true,
          ipv6: isIPV6,
          ip: clientIp,
          country_code: country_code
        };

        response.end(JSON.stringify(json, null, 2));
      } else {
        switch(params.query.field){
          case "ip":
            response.end(clientIp);
            break;

          case "country_code":
            response.end(country_code);
            break;

          default:
            response.end(clientIp);
            break;
        }
      }
    });
  };

  static handleNotFound = (request, response,  params, options) => {
  };
}

export default Api;
