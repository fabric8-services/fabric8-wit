Jenkins
=======

This directory and its content is meant to be used when building
this project on a CI system.

Every build dependency is covered by the `Dockerfile` which
essentially makes `docker` and `make` the only programs you need
to run the tests that our CI system is going to execute.

The `Makefile` has various targets and by default on the CI
system we run just `make` which is short for `make all` in our
`Makefile`.

If you want to know what eaxh target does you can run
`make help` to get a list of all targets and their description.

