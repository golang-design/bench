To configure systemd to run `bench`, run

```
$ sudo install -m 0644 bench.service /etc/systemd/system
$ sudo systemctl enable --now bench.service
```