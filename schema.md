# Database schema file

### URLs

**db.urls** - information about URLs

```js
{
  "_id": "short url",               // short URL
  "active": true,                   // link is active
  "prj": "Project2",                // project's name
  "orig": "origin URL",             // origin URL
  "u": "User1",                     // author of this link
  "ttl": ISODate(),                 // link's TTL
  "ndr": false,                     // no direct redirect
  "spam": 0.5,                      // smap coefficient
  "ts": ISODate()                   // date of creation
  "mod": ISODate()                  // date of modification
  "cb": ["GET", "http://a.ru", "p"] // custom callback settings
}

db.urls.ensureIndex({"_id": 1, "active": 1}, {"unique": 1})
db.urls.ensureIndex({"prj": 1, "active": 1, "u": 1})
db.urls.ensureIndex({"ttl": 1, "active": 1})
```

**db.ustats** - information about URLs statistics

```js
{
  "_id": ObjectId(),                     // item ID
  "url": "short url",                    // short URL
  "day": ISODate("2014-08-13 00:00:00")  // date (daily around)
  "c": 395                               // daily counter
}

db.ustats.ensureIndex({"url": 1, "day": 1}, {"unique": 1})
```

### Locks

**db.locks** - collection to control common locks

```js
{
  "_id": "urls",                   // locked collection
  "locked": false,                 // mutex flag
}

// test collection is used during test ping.
db.test.createIndex({"ts": 1 }, {expireAfterSeconds: 60})
```

### Projects

**db.projects** - information about projects and users.

```js
{
  "_id": ObjectId(),               // record ID
  "name": "Project1"               // project's name
  "domain": "http://domain.com",   // custom domain
  "users": [                       // info about users
    {
      "user": "User1",             // user's name
      "key": "sercrete token",     // secrete token
      "role": "owner",             // user's role
      "ts": Date()                 // date of modification
    },
    {
      "user": "User2",
      "key": "sercrete token",
      "role": "writer",
      "ts": Date()
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

db.projects.ensureIndex({"name": 1}, {"unique": 1})
db.projects.ensureIndex({"users.key": 1}, {"unique": 1})
```