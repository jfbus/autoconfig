# config
Autonomous configuration for package - initial config + hot reload

## Usage

```go
config.Load(ini.New(cfgfile))
config.ReloadOn(syscall.SIGHUP)
```

Package config :

```go
type PkgConf struct {
	Value string
}

func (c *PkgConf) Changed() {
	// Do something
}

var (
	pkfCong PkgConf{
		Value: "default value",
	}
	_ = config.Register("section name", &pkgConf)
)
```

Instance config :

```go
type PkgConf struct {
	Value string
}

var (
	// Set defaults
	_ = config.Register("section name", &PkgConf{
		Value: "default value",
	})
)

type PkgClass struct {}

func New() *PkgClass {
	n := &PkgClass{}
	config.Reconfigure("section name", n)
	return n
}

func (c *PkgClass) Reconfigure(c interface{}) {
	if cfg, ok := c.(PkgConf); ok {
		// Do something
	}
}
```

## Config file format

Any config file format can be used, provided :

* a loader class implementing the `Logger` interface is provided
```go
type Loader interface {
	Load(map[string]interface{}) error
}
```
* the corresponding library is able to unmarshal configs to `map[string]interface{}` where `interface{}` is a pointer to config structs.

Currently, loaders are available for :
* INI files (using https://github.com/go-ini/ini)

## License

MIT - see LICENSE