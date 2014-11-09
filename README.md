## mess

Mess is a new kind of social MUD. Written in Go, messes support:

* editing game things over the web
* making things interactive with Lua scripts
* talking with non-Latin scripts & emoji (in clients that support UTF-8)


### Building

Mess is still in development. To build it, install Go 1.3 & clone this repository. Then use the makefile:

    $ make env
    $ export GOPATH=`pwd`/env
    $ make assetsdev
    $ make mess

This creates a Go environment under the cloned repository in `./env`, downloads the required Go packages as imported by mess, and builds the current version of the mess server to `env/bin/mess`. You can then run it:

    $ env/bin/mess
