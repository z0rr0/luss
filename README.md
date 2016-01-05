# LUSS

**L**USS is a **U**RL **S**hortening **S**ervice.

**IT IS NOT READY YET!**

Default functionality:

* use RESTFull requests/responses
* get a short link as a response to the HTTP POST request

Special:

* multi-handler, it returns many short links for many incoming ones.
* callbacks, each short link can have a callback method with custom parameters
* link TTL (time to live), a short link can be temporary
* allow only authenticated clients

### API

User's token should be sent inside header **Authorization** with a prefix **Bearer**. If the token is not presented then request will be handled as anonymous one.

JSON array size has a limit, it is **maxpack** from the configuration file.

**JSON POST /api/add** - add new short links

```js
// request
[
  {
    "url": "http://some_url.com",
    "tag": "url tag",
    "ttl": 24,
    "nd": false,
    "group": "group #1",
    "cb": {
      "url": "http://callback_url.com",
      "method": "POST",
      "name": "param_name",
      "value": "param_value",
    }
  }
]

// response
{
  "errcode": 0,
  "msg": "ok",
  "result": [
    {
      "url": "http://some_url.com",
      "short": "http://short_url.com",
      "id": "short_url.com",
    }
  ]
}

// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"url": "http://domain", "tag": "", "group": "", "ttl": null, "nd": false, "cb": {"url": "", "method": "", "name": "", "value": ""}}]' http://<CUSTOM_DOMAIN>/api/add
```

**JSON POST /api/get** - get short links

```js
// request
[
  {
    "short": "http://short_url.com"
  }
]

// response
{
  "errcode": 0,
  "msg": "ok",
  "result": [
    {
      "url": "http://some_url.com",
      "short": "http://short_url.com",
      "id": "short_url.com",
    }
  ]
}

// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"short": "http://<CUSTOM_DOMAIN>/P"}, {"short": "http://<CUSTOM_DOMAIN>/O"}]' http://<CUSTOM_DOMAIN>/api/get
```

**JSON POST /api/user/add** - creates new user, only user with "admin" role has permissions for this request.

```js
// request
[
  {
    "name": "username"
  }
]

// response
{
  "errcode": 0,
  "msg": "ok",
  result: [
    {
      "name": "username",
      "token": "secrete token",
      "error": ""
    }
  ]
}

// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"name": "user1"}, {"name": "user2"}]' http://<CUSTOM_DOMAIN>/api/user/add
```

**JSON POST /api/user/pwd** - updates new user's token, only admin can change data of other users, but everyone can update his token.

```js
// request
[
  {
    "name": "username"
  }
]

// response
{
  "errcode": 0,
  "msg": "ok",
  result: [
    {
      "name": "username",
      "token": "secrete token",
      "error": ""
    }
  ]
}

// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"name": "user1"}, {"name": "user2"}]' http://<CUSTOM_DOMAIN>/api/user/pwd
```

**JSON POST /api/user/del** - remove users, only admin can admin has permissions for this request.

```js
// request
[
  {
    "name": "username"
  }
]

// response
{
  "errcode": 0,
  "msg": "ok",
  result: [
    {
      "name": "username",
      "error": ""
    }
  ]
}

// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"name": "user1"}, {"name": "user2"}]' http://<CUSTOM_DOMAIN>/api/user/del
```


### License

This source code is governed by a [LGPLv3](https://www.gnu.org/licenses/lgpl-3.0.txt) license that can be found in the [LICENSE](https://github.com/z0rr0/luss/blob/master/LICENSE) file.

<img src="https://www.gnu.org/graphics/lgplv3-147x51.png" title="LGPLv3 logo">


### ToDo

* email notifications
* export & import by admin
* projects add/edit
* projects approve by admin
* API requests
* projects statistics
* statistics