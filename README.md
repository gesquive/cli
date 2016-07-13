# cli-log

I consistently needed a simple output printer for my console projects that was simple and did not require a lot of setup. Since I couldn't find a package like this, I just wrote this quickly for myself.

## Usage

#### example.go

```
package main

import log "github.com/gesquive/cli-log"

func main() {
    SetLogLevel(LevelInfo)
	Debugln("debug")
	Infoln("info")
	Warnln("warn")
	Errorln("error")
	Fatalln("fatal")
}
```

```
info
warn
error
fatal
```

## License

This library is made available under an MIT-style license. See LICENSE.

## Contributing

PRs are always welcome!
