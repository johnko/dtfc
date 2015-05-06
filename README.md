# dtfc
DisTributed File Collection (because it's not great for CouchDB to store all doc.attachments in a single db file)

In CouchDB, all doc.attachments from a database were basically being concatenated into the single backend db file.

The backend db file increases in size forever until compacted. A large db file will take much longer to compact (as I think the compaction process copies leaf nodes to a new db file).

A db of large files, like photos and videos for example, would be painful to maintain.

## THOUGHTS

- Flags:
  - Listen on IP:PORT
  - Storage path
  - Peer list (file? dynamic?)
  - filter?
  - couchdb on localhost? to save map of sha512/filename
- Permissions:
  - rwx like Unix 777, except it's GET, PUT, DELETE
  - if file named "404" exists, return HTTP.404
  - if file named "403" exists, return HTTP.403
- Throttling:
  - Firewall max-conn-per-ip?
  - Token from CouchDB session?
- Storage:
  - storage path + sha512 hash (split every 2 chars) + "data"
  - save whole file as is in "data"
  - map file name to sha512
  - if possible keep the Storage path on a separate mount so you can run "df -h"
  - Verify sha512 before saving.
- In Response to:
  - PUT /filename: JSON sha512
  - GET /health: 200 OK
  - GET /sha512: data
  - DELETE /sha512: ???
- Proxy:
  - NginX for SSL & round robin load balancing

## HOW

- leverage groupcache, but instead of keeping the file in RAM, save it to local disk.
- client requests a file from this node, if this node doesn't have it on disk, ask peers for it, serve it and save it to local disk.

## SECURITY

- groupcache uses HTTP, you should only communicate with peers over a secure channel, and segregate this HTTP traffic from other internal network traffic.

## INSPIRATION FROM

- Apache [CouchDB](http://couchdb.apache.org/)
- my experience customizing [transfer.sh](https://transfer.sh/) into [wtfc](https://github.com/johnko/wtfc/)
- [groupcache](https://github.com/golang/groupcache)
- Trying to find an easier way than Amazon S3, Riak, RiakCS, GridFS, etc.

## LICENSE

Copyright (c) 2015 John Ko, Released under the MIT license
