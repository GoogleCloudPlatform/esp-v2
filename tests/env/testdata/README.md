# Test Data Guide

## Server cert

To regenerate: Edit `cert.conf`. Then run:

```sh
openssl req -nodes -new -x509 -sha256 -days 3600 -config cert.conf -extensions 'req_ext' -key server.key -out server.crt
```