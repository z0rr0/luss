# Database schema file

### URLs

**db.urls** - information about URLs

```js
{
  "_id": 123,                       // short URL and decimal number
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
  "users": [                       // info about users
    {
      "name": "User1",             // user's name
      "key": "sercrete token",     // secrete token
      "role": "owner",             // user's role
      "ts": Date()                 // date of modification
    },
    {
      "name": "User2",
      "key": "sercrete token",
      "role": "writer",
      "ts": Date()
    },
  ],
  "modified": ISODate(),           // date of modification
}

db.projects.ensureIndex({"name": 1}, {"unique": 1})
db.projects.ensureIndex({"users.key": 1}, {"unique": 1})
db.projects.ensureIndex({"users.name": 1})
```