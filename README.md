# Check My DNS Client

This [Go](https://golang.org) program is a command line interface for
[Check My DNS](https://cmdns.dev.dns-oarc.net) and will test the system
configured DNS resolver. All output on stdout is streamed JSON, each
object is separated with a new line. Status and errors are outputted on
stderr.

Use CTRL-C to break the program when it's done (or `-done`, see `-help`),
it does not exit on it's own because you can still get results after all
checks are done.

```shell
make dep
make
./cmdns-cli -help
```
