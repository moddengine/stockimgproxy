# Stock Image Proxy Service

Proxies multiple free stock image services into a single endpoint

Running container needs access the the following files:

 - `conf/config.json` Configuration File
 - `data/cache.db` Database for users & cache
 - `sock/fcgi.sock` FastCGI socket


### Configuration Template:

```json
{
  "pexels.com": {
    "key": "pexels api key - leave bank to skip pexels"
  },
  "unsplash.com": {
    "access": "public key - leave blank to skip"
  },
  "pixabay.com": {
    "key": "api key - leave blank to skip"
  },
  "debug": {
    "prettyJson": false
  }
}
```

### Authentication

HTTP Basic Authethentication

Use HTTP header
`Authorization: Basic xxxx`

`xxx` base64 encode `user:pass`

You will need to add a user to auth with:

```
sqlite3 data/cache.db

INSERT INTO users VALUES("username","argon2id hash of password", 1)

```

