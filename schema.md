# Database schema file

### Users

**db.users** - information about users

```js
{
  "_id": ObjectId(),               // user's ID
  "name": "username",              // user's name (max 256)
  "token": "123abc",               // user's secrete key (max 512)
  "role": "admin",                 // user's global role
  "created": ISODate()             // date of creation
}

db.users.ensureIndex({"name": 1}, {"unique": 1})
db.users.ensureIndex({"token": 1, "role": 1})
```

**db.keys** - application keys

```js
{
  "_id": ObjectId(),               // item's ID
  "value": "some key value",       // secrete app key
  "created": ISODate()             // date of creation
}
```


**db.urls** - information about URLs

```js
{
  "_id": "short url",              // short URL
  "prj": "Project2",               // project's name
  "orig": "origin URL",            // origin URL
  "req": 220,                      // requests' counter
  "author": "User1",               // author of this link
  "created": ISODate()             // date of creation
}
```

### Projects

**db.projects** - information about projects

```js
{
  "_id": ObjectId(),               // record ID
  "name": "Project1"               // project's name
  "domain": "http://somename.com", // custom domain
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
      "param": {}                  // custom parameters
    },
  ],
  "modified": ISODate(),            // date of modification
  "created": ISODate()             // date of creation
}
```