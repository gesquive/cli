# cli

I consistently needed a simple output printer for my console projects that did not require a lot of setup. Since I couldn't find a library to fill this role, I just wrote this project quickly for myself.

Features include:
 - cross-platform colored output
 - automatic tty detection
 - leveled logging

This library is not meant to be a comprehensive logging library. If you need a more comprehensive logging library, I highly recommend [Logrus](https://github.com/Sirupsen/logrus)

## Usage

#### example.go

```go
package main

import "github.com/gesquive/cli"

func main() {
	cli.SetLogLevel(cli.LevelInfo)
	cli.Debug("debug")
	cli.Info("info")
	cli.Warn("warn")
	cli.Error("error")
}
```

```
debug
info
warn
error
```

## License

This library is made available under an MIT-style license. See LICENSE.

## Contributing

PRs are always welcome!
