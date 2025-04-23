# go-storage

Fork from [Goravel](https://github.com/goravel/framework) for single use by necessary.

- Local Storage
- Minio
- s3

### Install

```shell
go get guthub.com/levskiy0/go-storage 
```

### Usage

```go 
import "github.com/owles/go-storage"

FS().
    AddDriver("local", NewLocal(LocalConfig{
        Root: "./storage",
        Url:  "https://test.io",
    })).
    SetDefaultDisk("local")

FS().Put("test.txt", "test data")

// Or

FS().Disk("minio_disk_name").Put("test.txt", "test data")

```
