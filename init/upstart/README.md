To configure Upstart to run `bench`, copy `bench.conf` into `/etc/init/` and run sudo start `bench`. E.g.,

```sh
$ sudo install -m 0644 bench.conf /etc/init/
$ sudo start bench
```