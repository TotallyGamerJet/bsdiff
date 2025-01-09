bsdiff / bspatch
==============
[![GoDoc](https://godoc.org/github.com/totallygamerjet/bsdiff?status.svg)](https://godoc.org/github.com/totallygamerjet/bsdiff)

Pure Go implementation of [bsdiff 4](http://www.daemonology.net/bsdiff/). Which provides a library for building and applying patches to binary
files.

The original algorithm and implementation was developed by Colin Percival.  The
algorithm is detailed in his paper, [Naïve Differences of Executable Code](http://www.daemonology.net/papers/bsdiff.pdf).  For more information, visit his
website at <http://www.daemonology.net/bsdiff/>.

## Install

```shell
go install -v github.com/totallygamerjet/bsdiff/cmd/...

bsdiff oldfile newfile patch
bspatch oldfile newfile2 patch
```

License
-------
Copyright 2003-2005 Colin Percival

Copyright 2019 Gabriel Ochsenhofer

Copyright 2025 TotallyGamerJet

This project is governed by the MIT license. For details see the file
titled LICENSE in the project root folder.