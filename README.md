# LUSS

**L**USS is a **U**RL **S**hortening **S**ervice.

It is a tool to reduce URLs length.

How it works - the service saves a custom URL and returns its short alias that redirects all incoming requests to the original web page.

Features:

* can be easy distributed using common database
* can handle anonymous or authenticated requests
* can track redirection requests (using GeoIP info)
* supports callbacks after redirections
* supports TTL (time to live) for temporary links
* supports cache control
* has RESTFull API: multi-items, users control

### API

Please read **[api.md](api.md)** file.


### License

This source code is governed by a [LGPLv3](https://www.gnu.org/licenses/lgpl-3.0.txt) license that can be found in the [LICENSE](https://github.com/z0rr0/luss/blob/master/LICENSE) file.

<img src="https://www.gnu.org/graphics/lgplv3-147x51.png" title="LGPLv3 logo">
