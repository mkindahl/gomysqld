Package `mysqld`
================

Library for working with MySQL server instances from within a Go
program. Note that this is not a MySQL *client* program; it is not
used to connect to MySQL servers. You have to use some other library
for that.

The library is distributed under the BSD License. See license text
below.

You can use the `go get` command to download and install the package
from from github.com:

    go get github.com/mkindahl/mysqld


Description
-----------

The library allow you to create and bootstrap new scatch instances of
MySQL servers, run some quick tests or programs, and then remove the
servers. It also allow you to manage different versions of servers in
the same setup so that you can test, for example, replication between
different versions of servers or run test scripts against different
versions of servers to check how they behave.

The package contains an API to work with distributions and servers as
well as a command-line utility using this API. You will see the
command-line interface here; if you're interested in the API, you
should look into the package documentation.


### Creating a MySQL Stable

To work with a set of servers, you need to create a stable where all
the distributions and servers will reside.

    gomysql init .

The information will be stored under the `.stable` directory in the
provided directory.


### Adding Distributions

Once you have a stable set up, you can add distributions to it. Each
distribution is installed from a binary distribution of the MySQL
server. If you want to install your own version, you need to build a
binary distribution from the source tree and add it to the stable
using the command:

    gomysql add distribution mysql-5.1.71-linux-x86_64-glibc23.tar.gz

If the distribution is an archive, the binary distribution will be
copied into the stable directory, but if a directory is given, a soft
link will be created in the stable.


### Working with Servers

Servers are created from distributions and you can create as many
servers as you like. When creating a server, a distribution name need
to be provided. However, since distribution names can be quite long,
it is sufficient to provide a unambigous substring of the distribution
name.

    gomysql add server my_server 5.1.71-linux

Once you are done with the server, you can remove it using:

    gomysql remove server my_server


Developer Notes
---------------

If you want to work with the code, there are a few suggestions in this
section.

### Running tests

Some tests require a distribution to execute. For those tests, a
distribution can be provided using the `-dist` flag. For example, to
run the tests using `mysql-5.6.14-linux-glibc2.5-i686.tar.gz`, provide
it with the `-dist` flag. Some tests require an expected version, so
to provide an expected version of the server to be found in the
distribution, use the `-version` flag:

    go test -dist=mysql-5.6.14-linux-glibc2.5-i686.tar.gz -version=5.6.14

Tests that require a distribution or an expected version to work will
be skipped if no distribution or expected version is provided.

Normally the tests remove the stable after being executed, but if you
want to debug the problem and check the stable that were created, you
can prevent it from being removed by using the flag `-keep`:

    go test -keep=true
