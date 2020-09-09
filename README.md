coredns-ipindn
==============
ipindn (IP-in-domain-name) is a plugin for CoreDNS.

This plugin is inspired by https://github.com/Eun/coredns-ipecho.


TLDR
----
A quick example to illustrate:

```bash
nslookup 182-32-22-12.example1.com localhost:5353
# Non-authoritative answer:
# Address: 182.32.22.12

nslookup foo.bar.182-32-22-12.example1.com localhost:5353
# Non-authoritative answer:
# Address: 182.32.22.12
```

IPv6 works too:

```bash
nslookup 1-db8-3c4d-15-0-0-1a2f-1a2b.example1.com localhost:5353
# Non-authoritative answer:
# Address: 1:db8:3c4d:15::1a2f:1a2b
```


Build Instructions
------------------
Git clone github.com/coredns/coredns.

Edit `plugin.cfg`. Insert somewhere between `auto:auto` and `secondary:secondary`: 

```
ipindn:github.com/gozl/coredns-ipindn/ipindn
```

Compile program. You need golang SDK:

```bash
go clean
go generate coredns.go
git_commit=$(git describe --dirty --always)
CGO_ENABLED=0 go build -v -ldflags="-s -w -X github.com/coredns/coredns/coremain.GitCommit=${git_commit}"
./coredns -version
```


Usage
-----
Edit `Corefile` to enable this plugin:

```
ipindn ZONE1 [ZONES...] {
    ttl DURATION
    fallthrough [ZONES...]
}
```

At least 1 authoritive zone must be specified. You can specify multiple zones.

TTL is in seconds. Defaults to 30.

You can specify zones that can fallthrough to subsequent plugins. Omitting [ZONES...] means all zones will fallthrough.

Here's a working example:

```
dns://.:5353 {
    ipindn example.com {
        ttl 30
        fallthrough
    }
}
```