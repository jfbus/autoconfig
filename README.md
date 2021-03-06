# autoconfig  [![Build Status](https://travis-ci.org/jfbus/autoconfig.svg)](https://travis-ci.org/jfbus/autoconfig) [![](https://godoc.org/github.com/jfbus/autoconfig?status.svg)](http://godoc.org/github.com/jfbus/autoconfig)

Autonomous configuration for golang packages with hot reload.

* Each package has its own configuration section within a single global config file, neither `main()` nor any other part of your application has the knowledge of the package configuration.
* Config can be dynamically updated when the application receives a signal.

Supported file format are :

* INI (using https://github.com/go-ini/ini)
* YAML (using https://gopkg.in/yaml.v2)

## Usage (YAML)

Init :

```go
autoconfig.Load(yaml.New(filename))
autoconfig.ReloadOn(syscall.SIGHUP)
```

Sample config file :

```yaml
[...]

section_name:
  group:
    value: foobar

[...]
```

Package config :

```go
package mypackage

type GroupConfig struct {
	Value `yaml:"value"`
}

type PkgConf struct {
	Group GroupConfig `yaml:"group"`
}

func (c *PkgConf) Changed() {
	// Do something when config has changed
}

var (
  // config, with default values
	pkfCong = PkgConf{
		Group: GroupConfig{
			Value: "default value",
		},
	}
	_ = autoconfig.Register("section_name", &pkgConf)
)
```

Instance config :

```go
package mypackage

var (
	// Set defaults
	_ = autoconfig.Register("section_name", &PkgConf{
		Group: GroupConfig{
			Value: "default value",
		},
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
		// Do something when config has changed
	}
}
```

_autoconfig will cleanly Lock/Unlock your structs provided they implement `sync.Locker`_


## Usage (INI)

Init :

```go
autoconfig.Load(ini.New(filename))
autoconfig.ReloadOn(syscall.SIGHUP)
```

Sample config file :

```ini
[...]

[section_name]
value=foobar

[...]
```

Package config :

```go
package mypackage

type PkgConf struct {
	Value string `ini:"value"`
}

var (
	pkfCong = PkgConf{
		Value: "default value",
	}
	_ = autoconfig.Register("section_name", &pkgConf)
)
```

### Other file formats

Any config file format can be used, provided a loader class implementing the `Loader` interface is provided :

```go
type Loader interface {
	Load(map[string]interface{}) error
}
```

## Caveats

* Only a single config file is supported,
* Values types are supported only if the underlying format supports them (e.g. INI does not support slices).

## TODO

* Multiple files

## License

MIT - see LICENSE
