{
  "listener": {
    "host": "",                       // HTTP server host
    "port": 10001,                    // HTTP server port
    "timeout": 30                     // HTTP server timeout
  },
  "database": {                       // MongoDB configuration:
    "hosts": ["localhost"],           //   servers list
    "port": 27017,                    //   connection port
    "timeout": 20,                    //   connection timeout (seconds)
    "username": "test",               //   auth username
    "password": "test",               //   auth password
    "database": "luss",               //   database name
    "authdb": "admin",                //   auth database
    "replica": null,                  //   replicaset name
    "ssl": false,                     //   use SSL connection
    "sslkeyfile": "",                 //   SSL key file
    "primaryread": true,              //   prefer primary read option
    "reconnects": 3,                  //   number of reconnects attempts (min 1)
    "rcntime": 50,                    //   delay between reconnects attempts (milliseconds)
    "debug": false                    //   debug mode
  },
  "cache": {                          // cache settings
    "dbpoolsize": 2,                  // size of database connections pool
    "dbpoolttl": 5,                   // period between pool clean (seconds)
  }
}