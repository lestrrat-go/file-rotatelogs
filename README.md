go-file-rotatelogs
==================

[![Build Status](https://travis-ci.org/lestrrat/go-file-rotatelogs.png?branch=master)](https://travis-ci.org/lestrrat/go-file-rotatelogs)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-file-rotatelogs?status.svg)](https://godoc.org/github.com/lestrrat/go-file-rotatelogs)


Port of [File::RotateLogs](https://metacpan.org/release/File-RotateLogs) from Perl to Go.

# SYNOPSIS

```go
import (
  "net/http"

  apachelog "github.com/lestrrat/go-apache-logformat"
  rotatelogs "github.com/lestrrat/go-file-rotatelogs"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { ... })

  logf := rotatelogs.New(
    "/path/to/access_log.%Y%m%d%H%M",
    rotatelogs.WithLinkName("/path/to/access_log"),
    rotatelogs.WithMaxAge(24 * time.Hour),
    rotatelogs.WithRotationTime(time.Hour),
  )

  http.ListenAndServe(":8080", apachelog.Wrap(mux, logf))
}
```

# DESCRIPTION

When you integrate this to to you app, it automatically write to logs that
are rotated from within the app: No more disk-full alerts because you forgot
to setup logrotate!

To install, simply issue a `go get`:

```
go get github.com/lestrrat/go-file-rotatelogs
```

It's normally expected that this library is used with some other
logging service, such as the built-in `log` library, or loggers
such as `github.com/lestrrat/go-apache-logformat`.

```go
import(
  "log"
  "github.com/lestrrat/go-file-rotatelogs"
)
  
func main() {
  rl := rotatelogs.NewRotateLogs("/path/to/access_log.%Y%m%d%H%M")

  log.SetOutput(rl)

  /* elsewhere ... */
  log.Printf("Hello, World!")
}
```

OPTIONS
====

## Pattern (Required)

The pattern used to generate actual log file names. You should use patterns
using the strftime (3) format. For example:

```go
  rotatelogs.New("/var/log/myapp/log.%Y%m%d")
```

## Clock (default: rotatelogs.Local)

You may specify an object that implements the roatatelogs.Clock interface.
When this option is supplied, it's used to determine the current time to
base all of the calculations on. For example, if you want to base your
calculations in UTC, you may specify rotatelogs.UTC

```go
  rotatelogs.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelogs.WithClock(rotatelogs.UTC),
  )
```

## LinkName (default: "")

Path where a symlink for the actual log file is placed. This allows you to 
always check at the same location for log files even if the logs were rotated

```go
  rotatelogs.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelogs.WithLinkName("/var/log/myapp/current"),
  )
```

```
  // Else where
  $ tail -f /var/log/myapp/current
```

If not provided, no link will be written.

## RotationTime (default: 86400 sec)

Interval between file rotation. By default logs are rotated every 86400 seconds.
Note: Remember to use time.Duration values.

```go
  // Rotate every hour
  rotatelogs.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelogs.WithRotationTime(time.Hour),
  )
```

## MaxAge (default: 0)

Time to wait until old logs are purged. By default no logs are purged, which
certainly isn't what you want.
Note: Remember to use time.Duration values.

```go
  // Purge logs older than 1 hour
  rotatelogs.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelogs.WithMaxAge(time.Hour),
  )
```
