+++
type = "post"
title = "Guide: Setting up a mint"
description = "Step by step guide to setting up a mint."
date = "2017-01-11T21:30:00-07:00"
+++

This article describe a step-by-step guide to setting up a mint. It was
conceived as I was setting up a QA mint at *t.settle.network* for testing
purposes.

# Setup a server

The first obvious step is to setup a server or use on of your choice on your
favorite cloud provider and get access to it over SSH. I'll assume the server
runs linux and you have access to it as a *sudoer* in the rest of the guide.

I recommand you create a user to run your services:
```
~$ sudo adduser mint
```

The *mint* user should not be able to `sudo`, in the rest of the guide, we'll
use the `mint:~$` prefix in bash commands supposed to be issued by the *mint*
user and `~$` for bash commands that require `sudo`.

# Installing Go

To run a mint, you'll need to install Go locally. The process is pretty
straightforward:

```
~$ wget https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz
~$ sudo tar -C /usr/local -xzf go1.7.4.linux-amd64.tar.gz
```

You then just need to setup *$GOROOT* and *$GOPATH*. As user *mint*, edit your
`/home/mint/.bashrc` file by adding the following at the ned of it:

```
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

Finally, as user *mint*, source your `.bashrc`, create your `~/go` directory,
and verify that your go installation is working:

```
mint:~$ . ~/.bashsrc
mint:~$ mkdir ~/go
mint:~$ go version
```

# Getting and building the `mint` executable

Downloading and building the `mint` executable as user *mint* is now as simple
as:

```
mint:~$ go get -u github.com/spolu/settle/...
```

The `mint` executable will be built and placed in `~/go/bin/mint` and should be
in your *$PATH*. You can check that the installation work by running:

```
mint:~$ mint --help
```

# Setting up a DNS

Even in QA your mint needs to be publicly addressable by other mints, so you'll
need to setup a DNS for your machine. *t.settle.network* uses AWS Route 53, but
any other domain name server from any regular registrar will work. You should
point the domain name for your mint to your machine as a CNAME or A record.
Here's the example DNS record for *t.settle.network* in the *settle.network*
zone file:

```
t 600 IN CNAME ec2-35-162-152-151.us-west-2.compute.amazonaws.com.
```

# Setting up HAProxy

I'd recommand serving the traffic for your mint with HAProxy, it's robust, has
a ton of tooling and will help you setup TLS termination if you want to move
your mint in production. Install `haproxy` locally (on Ubuntu):

```
~$ sudo apt-get install haproxy
```

Edit the HAProxy configuraiton file with:

```
~$ sudo vim /etc/haproxy/haproxy.cfg
```

Here's the configuration file used by *t.settle.network*:
```
global
        log /dev/log    local0
        log /dev/log    local1 notice
        chroot /var/lib/haproxy
        stats socket /run/haproxy/admin.sock mode 660 level admin
        stats timeout 30s
        user haproxy
        group haproxy
        daemon
        ca-base /etc/ssl/certs
        crt-base /etc/ssl/private
        ssl-default-bind-ciphers ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:ECDH+3DES:DH+3DES:RSA+AESGCM:RSA+AES:RSA+3DES:!aNULL:!MD5:!DSS
        ssl-default-bind-options no-sslv3

defaults
        log     global
        mode    http
        option  httplog
        option  dontlognull
        timeout connect 5000
        timeout client  50000
        timeout server  50000

frontend frt-qa-mint
        bind *:2047
        reqadd X-Forwarded-Proto:\ http
        acl is_mint hdr(host) -i t.settle.network
        use_backend qa-mint if is_mint

backend qa-mint
        server qa-mint1 127.0.0.1:2047
```

QA mints have to listen on port *2407* and we'll be running it locally on the
same port.

# Running the mint in QA

You're now all set to run the mint. By default the mint will use SQLite3 and
store the DB file in `~/.mint/mint-qa.db`:

```
mint:~$ mint -env=qa -host=t.settle.network -port-2407
```

# Creating a user for your mint

The `mint` command provides a utility to add a user to the locally running mint:
```
mint:~$ mint -env=qa -action=create_user -username=spolu -password=...
```

## Testing your QA mint

From your local machine with `settle` installed you should now be able to run
`settle login -env=qa` and provide the address of the user you just created
(here *spolu@t.settle.network*) with the password you specified. You can now
interact with you QA mint using the `settle -env=qa` command (make sure to
always pass the `-env=qa` flag while testing).

# Moving to production (advanced)

Moving to production mainly consits in running the mint with the `-env=prod`
flag and setting up TLS termination with HAProxy.

For the production *m.settle.network* mint, we rely on free *Let's Encrypt* SSL
certificates. I'll refer you to the
[lego](https://github.com/letsencrypt/acme-spec) tool for more info on how to
do so copy-pasting the script we run periodically for renewal:

```
lego --email="polu.stanislas@gmail.com" --domains="m.settle.network" renew
cat m.settle.network.crt m.settle.network.key > /etc/ssl/private/m.settle.network.pem
```

As well as the HAProxy configuration for *m.settle.network*:

```
frontend frt-prod-mint
        bind *:2046 ssl crt /etc/ssl/private/m.settle.network.pem
        reqadd X-Forwarded-Proto:\ https
        acl is_mint hdr(host) -i m.settle.network
        use_backend prod-mint if is_mint

backend prod-mint
        server mint1 127.0.0.1:2046
```

## Graceful deploys

The `mint` executable supports graceful restarts. Whenever you want to upgrade
your mint deployment you simply need to run:

```
mint:~$ go get -u github.com/spolu/settle/...
mint:~$ killall -s USR2 mint

```

# Conclusion

I hope you managed to setup your mint successfully without too much trouble.
This guide is definitely quite "raw"!  The source of this post is available
publicly[0], please don't hesitate to submit pull-requests if you see any way
to improve it!

-stan

[0] https://github.com/spolu/settle/blob/master/site/content/posts/setting-up-a-mint.md
