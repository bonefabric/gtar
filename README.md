# Tar archiver

---

Simple tool to create and extract data from tar and tar.gz archives

---
#### Flags:

- f - tar archive file name
- x - extract from archive

### Examples:

Make tar 
```shell
gtar .
```

```shell
gtar -f archive.tar /root
gtar -f archive.tar.gz /mnt/
```

Extract tar
```shell
gtar -x -f archive.tar .
```

```shell
gtar -x -f archive.tar.gz /root/data
```