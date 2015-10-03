# Database schema file

### Users

**db.users** - information about users

```js
{
  "_id": ObjectId(),               // user's ID
  "name": "username",              // user's name (max 256)
  "token": "123abc",               // user's secrete key (max 512)
  "role": "admin",                 // user's global role
  "modified": ISODate(),           // date of modification
  "created": ISODate()             // date of creation
}

db.users.ensureIndex({"name": 1}, {"unique": 1})
db.users.ensureIndex({"token": 1, "role": 1})
```

**db.locks** - collection to control common locks

```js
{
  "_id": "urls",                   // locked collection
  "locked": false,                 // mutex flag
  "pid": "host-pid"                // some program's identifier
}

//db.locks.ensureIndex({"_id": 1, "locked": 1}, {"unique": 1})
db.test.createIndex({"ts": 1 }, {expireAfterSeconds: 60})
```

### URLs

**db.urls** - information about URLs

```js
{
  "_id": "short url",              // short URL
  "active": true,                  // link is active
  "prj": "Project2",               // project's name
  "orig": "origin URL",            // origin URL
  "u": "User1",                    // author of this link
  "ttl": ISODate(),                // link's TTL
  "ndr": false,                    // no direct redirect
  "spam": 0.5,                     // smap coefficient
  "ts": ISODate()                  // date of creation
}

db.urls.ensureIndex({"_id": 1, "active": 1}, {"unique": 1})
db.urls.ensureIndex({"prj": 1, "u": 1})
```

**db.ustats** - information about URLs statistics

```js
{
  "_id": ObjectId(),                     // item ID
  "url": "short url",                    // short URL
  "day": ISODate("2014-08-13 00:00:00")  // date (daily around)
  "c": 395                               // daily counter
}

db.urls.ensureIndex({"url": 1, "day": 1}, {"unique": 1})
```

### Projects

**db.projects** - information about projects

```js
{
  "_id": ObjectId(),               // record ID
  "name": "Project1"               // project's name
  "domain": "http://domain.com",   // custom domain
  "users": [                       // info about users
    {
      "user": "User1",             // user's name
      "key": "sercrete token",     // secrete token
      "role": "owner"              // user's role
    },
    {
      "user": "User2",
      "key": "sercrete token",
      "role": "writer"
    },
  ]
  "callbacks": [                   // callack methods
    {
      "method": "GET",             // request type
      "url": "http//callback.com", // callback url
      "params": {"a": 123}         // custom parameters
    },
  ],
  "modified": ISODate(),           // date of modification
  "created": ISODate()             // date of creation
}
```