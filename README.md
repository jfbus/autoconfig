# autoconfig  [![](https://godoc.org/github.com/jfbus/autoconfig?status.svg)](http://godoc.org/github.com/jfbus/autoconfig)

Autonomous configuration for golang packages with hot reload.

* Each package has its own configuration, neither `main()` nor any other part of your application has the knowledge of the package configuration. 
* Config can be dynamically updated when the application receives a signal.

## Usage

Init :

```go
autoconfig.Load(ini.New(cfgfile))
autoconfig.ReloadOn(syscall.SIGHUP)
```

Sample config file:

```ini
[section_name]
value=foobar
```

Package config :

```go
package mypackage

type PkgConf struct {
	Value string `ini:"value"`
}

func (c *PkgConf) Changed() {
	// Do something
}

var (
	pkfCong = PkgConf{
		Value: "default value",
	}
	_ = autoconfig.Register("section_name", &pkgConf)
)
```

Instance config :

```go
package mypackage

type PkgConf struct {
	Value string `ini:"value"`
}

var (
	// Set defaults
	_ = autoconfig.Register("section_name", &PkgConf{
		Value: "default value",
	})
)

type PkgClass struct {}

func New() *PkgClass {
	n := &PkgClass{}
	// This will trigger a n.Reconfigure() call with the current config
	autoconfig.Reconfigure("section_name", n)
	return n
}

func (c *PkgClass) Reconfigure(c interface{}) {
	if cfg, ok := c.(*PkgConf); ok {
		// Do something
	}
}
```

_autoconfig will cleanly Lock/Unlock your structs provided they implement `sync.Locker`_

## Config file formats

Any config file format can be used, provided :

* the corresponding library is able to unmarshal configs to `map[string]interface{}` (map of strings of pointers to structs).
* a loader class implementing the `Loader` interface is provided
```go
type Loader interface {
	Load(map[string]interface{}) error
}
```

Currently, loaders are available for :
* INI files (using https://github.com/go-ini/ini)

## Caveats

* Only a single file per config is supported,
* Slice values are currently not supported.

## Todo

* Slice support
* Tests

## License

MIT - see LICENSE