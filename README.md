# hashttp - An http server that serves content by its hash sum

This simple http server takes a bunch of files as command line arguments,
generates SHA512 hashes for each of them and serves them via http, using
the hash as as path in the request URI.

Apart from regular files, the code also handles raw Linux block devices.
Built-in support for basic content parsing allows for proper file size
determination.

Originally, the code was written to serve a squashfs image from one machine
to another as part of an update process. A client that knows the SHA512 can
then easily retrieve the contents from another machine. Considering the SHA512
secret, it also serves as an access control method.

# Building

This project is implemented in Go, and the build process is as simple as this:

`go build`

# Parameters

There are several command line parameters that can be passed:

* `--source` may be used several times to specify a local file to be scanned
  and served. At least one such file must be given. If multiple given files
  happen to have the same hash, only the first of them is used.

* `--port` specifies the port to listen to. Must be passed, the program does
  not default to a value.

* `--url-prefix` can be optionally used to pass a prefix to expect before the
  hash sum.

# File type handling

Block devices can be used as a source directly, in which case the size of the
device will be determined, and the entire content is read and served.

This, however, is not always what is desired. For instance, when file system
images are stored on a block device, only a fraction of the device actually
contains useful data. To support this, hashttp is prepared to parse the headers
of such content to determine the actual size that is of interest.

Currently, only squashfs images are detected during parsing, but more file
types can easily be added. Please do post PRs!

# Example

Let's create a file and see which SHA512 is has.

```
$ echo "Hello, world" >file.txt
$ sha512sum file.txt
959c0bdfa9877d3466c5848f55264f72f132c657b002b79fda65dbe36c67f4bb3d2a3e2e9925cb5896a53c76169c5bb71b7853bd90192068dc77f4b20159a1d8  file.txt
```

Now, let's start the server and point it to that file.

```
$ ./hashttp --source file.txt --port 5555 --url-prefix files                                                                                                                                                                (130) [14:56:01]
2019/01/05 14:56:03 Processing source file.txt ...
2019/01/05 14:56:03 Serving file.txt (type plain) on /files/959c0bdfa9877d3466c5848f55264f72f132c657b002b79fda65dbe36c67f4bb3d2a3e2e9925cb5896a53c76169c5bb71b7853bd90192068dc77f4b20159a1d8
2019/01/05 14:56:03 Listening on port 5555 ...
```

And then retrieve it again via http.

```
$ curl -o - http://localhost:5555/files/959c0bdfa9877d3466c5848f55264f72f132c657b002b79fda65dbe36c67f4bb3d2a3e2e9925cb5896a53c76169c5bb71b7853bd90192068dc77f4b20159a1d8                                         (2) [14:57:36]
Hello, world
```

# License

GPLv2, see the `LICENSE.GPL2` file.

