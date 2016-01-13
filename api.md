# LUSS API description

User's token should be sent inside header **Authorization** with a prefix **Bearer**. If the token is not presented then request will be handled as anonymous one.

The response after error:

```js
{
  "errcode": 404,         // error code
  "msg": "error message", // error description
  "result": []            // empty result array
}
```

JSON array size has a limit, it is **maxpack** from the configuration file.

**JSON GET /api/info** - get main service info

```js
// request
Not needed any data.

// response
{
  "errcode": 0,
  "msg": "ok",
  "result": [
    {
      "version": "0.0.1", // API version
      "authok": true,     // or false token is empty or invalid
      "pack_size": 512    // max request pack size
    }
  ]
}
```

```sh
// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" http://<CUSTOM_DOMAIN>/api/info
```

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
```

```sh
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
      "error": ""
    }
  ]
}
```

```sh
// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"short": "http://<CUSTOM_DOMAIN>/Pr"}, {"short": "http://<CUSTOM_DOMAIN>/Hw"}]' http://<CUSTOM_DOMAIN>/api/get
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
```

```sh
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
```

```sh
// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"name": "user1"}, {"name": "user2"}]' http://<CUSTOM_DOMAIN>/api/user/pwd
```

**JSON POST /api/user/del** - remove users, only admin can has permissions for this request.

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
```

```sh
// example
curl -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"name": "user1"}, {"name": "user2"}]' http://<CUSTOM_DOMAIN>/api/user/del
```

**JSON POST /api/import** - import other short URLs (only for admin)

```js
// request
[
  {
    "url": "http://some_url.com",
    "short": "short_url"
  }
]

// response
{
  "errcode": 0,
  "msg": "ok",
  result: [
    {
      "short": "short_url",
      "error": ""
    }
  ]
}

```

```sh
// example
curl -v -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"url": "http://some_url.com", "short": "ab"}]' http://<CUSTOM_DOMAIN>/api/import

```

**JSON POST /api/export** - export URLs data (only for admin)

```js
// request
{
  "group": "some_group",
  "tag": "some_tag",
  "period": ["2015-01-01", "2015-12-31"],
  "active": true,
  "page": 1
}

// response
{
  "errcode": 0,
  "msg": "ok",
  "pages": [1, 10, 100], // current, total, page_size
  "result": [
    {
      "id": "short_url",
      "short": "http://short_url.com",
      "url": "http://some_url.com",
      "group": "some_group",
      "tag": "some_tag",
      "created": "2015-06-30",
    }
  ]
}

```

```sh
// example
curl -v -H "Content-Type: application/json" -H "Authorization: Bearer<TOKEN>" -X POST --data '[{"group": "", "tag": "", "period": ["2015-01-01", ""], "active": true}]' http://<CUSTOM_DOMAIN>/api/export

```