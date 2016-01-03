# Database schema file

### URLs

**db.urls** - information about URLs

```js
{
  "_id": 123,                       // short URL and decimal number
  "off": false,                     // link is not active
  "group": "Group1",                // project's name
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
    "name": "name",                 //   callback parameter name
    "value": "string parameter",    //   callback parameter value (also _id and tag will be added)
  }
}

db.urls.ensureIndex({"group": 1, "off": 1, "u": 1})
db.urls.ensureIndex({"ttl": 1, "off": 1})
```

### Tracks

**db.tracks** - tracker collection

```js
{
  "_id": ObjectId(),                // item ID
  "short": "short url",             // short URL
  "url": "original url",            // original URL
  "group": "group name",            // project's name
  "tag": "tag1",                    // tag value
  "geo": {                          // geo IP information:
    "ip": "127.0.0.1",              //   IP address
    "country": "name",              //   country name
    "city": "name",                 //   city name
    "tz": "UTC"                     //   timezone
    "lat": 51.5142,                 //   latitude
    "lon": -0.0931                  //   longitude
  }
  "ts": ISODate()                   // created date
}

db.tracks.ensureIndex({"group": 1, "ts": 1})
```

### Locks

**db.locks** - collection to control common locks

```js
{
  "_id": "key",                    // locked key
}
```

### Users

**db.users** - information about users.

```js
{
  "_id": "name",                    // user's name (max 255)
  "off": false,                     // use is deactivated
  "token": "secrete token",         // secrete token
  "roles": ["root"],                // user's roles
  "ct": Date(),                     // created
  "mt": Date()                      // modified
}

db.users.ensureIndex({"token": 1}, {"unique": 1})
```

### Tests

**db.tests** - collection for test requests.

```js
{
  "_id": ObjectId,                  // item identifier
  "ts": Date()                      // timestamp
}
```