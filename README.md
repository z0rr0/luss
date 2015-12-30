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

### License

This source code is governed by a [LGPLv3](https://www.gnu.org/licenses/lgpl-3.0.txt) license that can be found in the [LICENSE](https://github.com/z0rr0/luss/blob/master/LICENSE) file.

<img src="https://www.gnu.org/graphics/lgplv3-147x51.png" title="LGPLv3 logo">


### ToDo

* requests tracker
* admin creation
* email notifications
* export & import by admin
* projects add/edit
* projects approve by admin
* API requests
* projects statistics
* callbacks
* statistics