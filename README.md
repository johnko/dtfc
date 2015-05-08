# dtfc (aka groupstore: a slow, persistent, alternative to groupcache)
DisTributed File Collection (because it's not great for CouchDB to store all doc.attachments in a single db file)

In CouchDB, all doc.attachments from a database were basically being concatenated into the single backend db file. The backend db file increases in size forever until compacted. A large db file will take much longer to compact (as I think the compaction process copies leaf nodes to a new db file). A db of large files, like photos and videos for example, would be painful to maintain.

I'm hoping dtfc can be a simple distributed GET/PUT service for CouchDB attachments.

## HOW

- ~~leverage groupcache, but~~
- instead of keeping the file in RAM, save it to local disk.
- client requests a file from this node, if this node doesn't have it on disk, ask peers for it, serve it and save it to local disk.

## BUILD DEPENDENCIES

see `scripts/load-dependencies.sh`

## RUN DEPENDENCIES

- shasum: `/usr/local/bin/shasum` or `/opt/local/bin/shasum`

- tail: `/usr/bin/tail`

## THOUGHTS

- Flags:
  - [x] --port PORT
  - [x] --basedir (Storage path)
  - [x] --me http://IP:PORT/
  - [x] --peerlist Peer list (file? dynamic?)
  - [ ] filtered replication?
  - [ ] couchdb on localhost? to save map of sha512/filename
- Permissions:
  - [x] rwx like Unix 777, except it's GET, PUT, DELETE
  - [ ] if file named "404" exists, return HTTP.404
  - [ ] if file named "403" exists, return HTTP.403
- Peering:
  - [x] sequential
  - [ ] better peer picker than sequential
- Uploading:
  - [ ] PUT to another peer during first upload to avoid SPOF?
- Throttling:
  - [ ] if not tracked by CouchDB, purge to prevent spam?
  - [ ] Firewall max-conn-per-ip?
  - [ ] Token from CouchDB session?
- Storage:
  - [x] storage path + sha512 hash (split every 2 chars) + "data"
  - [x] save whole file as is in "data"
  - [ ] map file name to sha512
  - [ ] if possible keep the Storage path on a separate mount so you can run "df -h"
- Integrity:
  - [x] Verify sha512 before saving (it's part of the save process)
  - [ ] Verify sha512 before serving
- In Response to:
  - [x] PUT /filename: JSON of sha512 and filename
  - [x] GET /health: 200 OK
  - [x] GET /sha512: data
  - [ ] DELETE /sha512: ???
- Security:
  - [ ] figure out how to do authentication/authorization
- Proxy:
  - [ ] NginX for SSL & round robin load balancing

## SECURITY

- do not run as root, create a non-priviledged user
- use a rate limiting firewall
- ~~groupcache uses HTTP, you should only communicate with peers over a secure channel, and~~
- segregate this HTTP traffic from other internal network traffic.

## INSPIRATION FROM

- Apache [CouchDB](http://couchdb.apache.org/)
- my experience customizing [transfer.sh](https://transfer.sh/) into [wtfc](https://github.com/johnko/wtfc/)
- [groupcache](https://github.com/golang/groupcache)
- Trying to find an easier way than Amazon S3, Riak, RiakCS, GridFS, etc.

## LICENSE

Copyright (c) 2015 John Ko, Released under the MIT license
