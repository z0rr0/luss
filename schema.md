# Database schema file

### URLs

**db.urls** - information about URLs

```js
{
  "_id": 123,                       // short URL and decimal number
  "off": false,                     // link is not active
  "prj": "Project2",                // project's name
  "tag": "tag1",                    // tag (some custom identifier)
  "orig": "origin URL",             // origin URL
  "u": "User1",                     // author of this link
  "ttl": ISODate(),                 // link's TTL
  "ndr": false,                     // no direct redirect
  "spam": 0.5,                      // smap coefficient
  "ts": ISODate()                   // date of creation
  "mod": ISODate()                  // date of modification
  "cb": {                           // callback settings
    "u": "https://domain.com/",     //   callback URL
    "m": "GET",                     //   callback method
    "p": "string parameter",        //   additional callback parameter (also _id and tag will be added)
  }
}

db.urls.ensureIndex({"prj": 1, "off": 1, "u": 1})
db.urls.ensureIndex({"ttl": 1, "off": 1})
```

**db.ustats** - information about URLs statistics

```js
{
  "_id": ObjectId(),                // item ID
  "url": "short url",               // short URL
  "tag": "tag",                     // link's tag (some custom identifier)
  "date": ISODate()                 // date (daily around)
}

db.ustats.ensureIndex({"url": 1, "day": 1}, {"unique": 1})
```

### Locks

**db.locks** - collection to control common locks

```js
{
  "_id": "urls",                    // locked collection
  "locked": false,                  // mutex flag
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
  "tags": ["tag1", "tag1"],        // custom project's tags
  "modified": ISODate(),           // date of modification
}

db.projects.ensureIndex({"name": 1}, {"unique": 1})
db.projects.ensureIndex({"users.key": 1}, {"unique": 1})
db.projects.ensureIndex({"users.name": 1})
```