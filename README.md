Super simple tool to create a SSL proxy in front of an http server.
The credit goes to the standard library for implementing the core of
what this provides.

simplehttpsproxy sets X-Forwarded-For header on the request to the backend.

$ go get github.com/yannk/simplehttpsproxy
$ go install github.com/yannk/simplehttpsproxy
$ simplehttpsproxy

- or

$ simplehttpsproxy -host=proxyhost.com -cert=cert.pem -key=key.pem -listen=localhost:4443 -backend:8080

See also https://github.com/yannk/simplehttpserver
