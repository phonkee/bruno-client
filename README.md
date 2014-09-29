Bruno Client
============

This is the client for [Bruno VoIP system](https://github.com/vhakulinen/bruno-server) and is even more WIP than the server.

With this client, you'll only be able to make voice call. Server support text messages but this client doesnt (atm).

Running it
==========

Change the IPs in the source code to your liking (if not running server on localhost) and just `go run client.go`.

When the client is running
```
>>> /register <username> <password>
>>> /login <username> <password>
>>> /udp_init
>>> /call <username>
```

To answer
```
>>> /answer
```

You'll need portaudio to make the calls.


Contribuing
===========

If you have ideas on peer-to-peer protocol, open up an issue and don't go working on it by your self without me (or us) knowing about it. Will make stuff easier.

At the moment (GitHub release) there is one or two unnecessary gorutines which can be removed (fixed) and interface is
shit. I'm new to Go so there might be some Go-idiom stuff too so feel free to fix those or alteast let me know about it.
