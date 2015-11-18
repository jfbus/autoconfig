/*
Package autoconfig allows packages to be configured autonomously and reconfigured automatically.

Each package has its own configuration section in a global config file, neither main() nor any other part of your application has the knowledge of the package configuration.
Config can be dynamically updated when the application receives a signal.


Supported file format are INI (using https://github.com/go-ini/ini) and YAML (using https://gopkg.in/yaml.v2).

Usage - YAML

Init :

	autoconfig.Load(yaml.New(cfgfile))
	autoconfig.ReloadOn(syscall.SIGHUP)

Sample config file :

	section_name:
		group:
			value: foobar

Package config :

	package mypackage

	type GroupConfig struct {
		Value `yaml:"value"`
	}

	type PkgConf struct {
		Group GroupConfig `yaml:"group"`
	}

	func (c *PkgConf) Changed() {
		// Do something
	}

	var (
		pkfCong = PkgConf{
			Group: GroupConfig{
				Value: "default value",
			},
		}
		_ = autoconfig.Register("section_name", &pkgConf)
	)

Instance config :

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
			// Do something
		}
	}

autoconfig will cleanly Lock/Unlock your structs provided they implement sync.Locker


Usage - INI

Init :

	autoconfig.Load(ini.New(cfgfile))
	autoconfig.ReloadOn(syscall.SIGHUP)

Sample config file :

	[section_name]
	value=foobar

Package config :

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


Other file formats

Any config file format can be used, provided a loader class implementing the `Loader` interface is provided :

	type Loader interface {
		Load(map[string]interface{}) error
	}

Caveats

* Only a single config file is supported,

* Values types are supported only if the underlying format supports them (e.g. INI does not support slices).

*/
package autoconfig

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sync"
)

type section struct {
	defaults  reflect.Value
	current   interface{}
	signature string
	onchange  []Reconfigurable
}

// Config defines a config
type Config struct {
	filename string
	sections map[string]*section
	current  map[string]interface{}
	loader   Loader
	loaded   bool
}

// UpdatableConfig defines the interface updateable config need to implement.
// Each time the config is reloaded and the corresponding config section has changed,
// the Changed function will be called.
type UpdatableConfig interface {
	Changed()
}

// Reconfigurable defines the interface updateable instances need to implement.
// Each time the config is reloaded and the corresponding config section has changed,
// the Reconfigure function will be called for all instances.
type Reconfigurable interface {
	Reconfigure(interface{})
}

// Loader defines the interface a config file loader will need to implement.
type Loader interface {
	Load(map[string]interface{}) error
}

var (
	globalConfig = Config{sections: map[string]*section{}, current: map[string]interface{}{}}

	ErrNoLoader = errors.New("No loader was defined")
)

// New defines a config, based on a loader.
func New(l Loader) *Config {
	return &Config{sections: map[string]*section{}, current: map[string]interface{}{}, loader: l}
}

// Load loads the config by calling the Load() function of the loader.
func (c *Config) Load() error {
	c.loaded = true
	return c.load()
}

// Load defines the loader for the default config, and loads the config file.
func Load(l Loader) error {
	globalConfig.loader = l
	return globalConfig.Load()
}

// Reload reloads the config file
func (c *Config) Reload() error {
	return c.Load()
}

// Reload reloads the config file for the default config
func Reload() error {
	return globalConfig.Reload()
}

// ReloadOn defines signal to monitor. On reception of a signal, the config will be reloaded.
func (c *Config) ReloadOn(signals ...os.Signal) {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, signals...)
		for _ = range ch {
			c.Reload()
		}
	}()
}

// ReloadOn defines signal to monitor. On reception of a signal, the default config will be reloaded.
func ReloadOn(signals ...os.Signal) {
	globalConfig.ReloadOn(signals...)
}

// Register registers a config structure for a config file section. The values passed will be used as
// defaults in the future.
// Defaults will be remembered : if a variable is defined, and then unset, it will be reset to the default value.
// If s implements UpdateableConfig, s.Changed() will be called when the config is reloaded and has changed.
// If config has been previously loaded, s.Changed() will be called immediatly.
func (c *Config) Register(name string, s interface{}) bool {
	if uc, ok := s.(UpdatableConfig); ok {
		c.register(name, s, &reconfigurableCfg{uc})
	} else {
		c.register(name, s, nil)
	}
	if c.loaded {
		c.Reload()
	}
	return true
}

// Register registers a config structure for a config file section. The values passed will be used as
// defaults in the future.
// Defaults will be remembered : if a variable is defined, and then unset, it will be reset to the default value.
// If s implements UpdateableConfig, s.Changed() will be called when the config is reloaded and has changed.
//
// 	var (
// 		_ = config.Register("section_name", &PkgConfig{Value: "default"})
// 	)
func Register(name string, s interface{}) bool {
	return globalConfig.Register(name, s)
}

// Reconfigure registers an instance. The config section must have been registered before using Register
// r.Reconfigure() will be called when config is reloaded and has changed.
// If config has been previously loaded, r.Reconfigure() will be called immediatly.
func (c *Config) Reconfigure(name string, r Reconfigurable) bool {
	c.register(name, nil, r)
	if c.loaded {
		if cfg, ok := c.Get(name); ok {
			r.Reconfigure(cfg)
		}
	}
	return true
}

// Reconfigure registers an instance to the default config. The config section must have been registered before using Register
//
// 	func New() *PkgClass {
// 		c := &PkgClass{}
// 		config.Reconfigure("section_name", c)
// 		return c
// 	}
func Reconfigure(name string, r Reconfigurable) bool {
	return globalConfig.Reconfigure(name, r)
}

// Get returns the configuration for a section
func (c *Config) Get(name string) (interface{}, bool) {
	cfg, ok := c.current[name]
	return cfg, ok
}

// Get returns the configuration for a section
func Get(name string) (interface{}, bool) {
	return globalConfig.Get(name)
}

// MustGet returns the configuration for the specified section. If the section does not exist, something will panic.
func (c *Config) MustGet(name string) interface{} {
	return c.current[name]
}

// MustGet returns the configuration for the specified section from the default configuration. If the section does not exist, something will panic.
func MustGet(name string) interface{} {
	return globalConfig.MustGet(name)
}

type reconfigurableCfg struct {
	c UpdatableConfig
}

func (r *reconfigurableCfg) Reconfigure(n interface{}) {
	r.c.Changed()
}

func (r *reconfigurableCfg) Lock() {
	if l, ok := r.c.(sync.Locker); ok {
		l.Lock()
	}
}

func (r *reconfigurableCfg) Unlock() {
	if l, ok := r.c.(sync.Locker); ok {
		l.Unlock()
	}
}

func (c *Config) register(name string, defaults interface{}, r Reconfigurable) {
	v := reflect.Indirect(reflect.ValueOf(defaults))
	if _, found := c.sections[name]; !found {
		c.sections[name] = &section{
			defaults: reflect.New(v.Type()),
			onchange: []Reconfigurable{},
		}
	}
	if defaults != nil {
		d := c.sections[name].defaults
		switch d.Type().Kind() {
		case reflect.Struct:
			addStructDefaults(d, v)
		case reflect.Map:
			addMapDefaults(d, v)
		default:
		}
		if c.sections[name].current == nil {
			c.sections[name].current = defaults
			c.current[name] = defaults
		}
	}
	if r != nil {
		c.sections[name].onchange = append(c.sections[name].onchange, r)
	}
}

func (c *Config) load() error {
	if c.loader == nil {
		return ErrNoLoader
	}
	for _, section := range c.sections {
		if l, ok := section.current.(sync.Locker); ok {
			l.Lock()
			defer l.Unlock()
		}
	}
	err := c.loader.Load(c.current)
	if err != nil {
		return err
	}
	for _, section := range c.sections {
		section.change()
	}
	return err
}

func (s *section) change() {
	sig, err := json.Marshal(s.current)
	if err != nil || string(sig) != s.signature {
		for _, r := range s.onchange {
			r.Reconfigure(s.current)
		}
		s.signature = string(sig)
	}
}

func addMapDefaults(to, from reflect.Value) {
	to = reflect.Indirect(to)
	from = reflect.Indirect(from)
	for _, key := range from.MapKeys() {
		f := to.MapIndex(key)
		if reflect.DeepEqual(f.Interface(), reflect.Zero(f.Type()).Interface()) {
			if !f.CanSet() {
				log.Printf("Config: Cannot set default value for key %s of %s", key, f.Type().Name())
				continue
			}
			docopy(f, from.MapIndex(key))
		}
	}
}

func addStructDefaults(to, from reflect.Value) {
	to = reflect.Indirect(to)
	from = reflect.Indirect(from)
	for i := 0; i < to.NumField(); i++ {
		f := to.Field(i)
		if to.Type().Field(i).PkgPath != "" {
			continue
		}
		if reflect.DeepEqual(f.Interface(), reflect.Zero(f.Type()).Interface()) {
			if !f.CanSet() {
				log.Printf("Config: Cannot set default value for field %s of %s", f.Type().Field(i).Name, f.Type().Name())
				continue
			}
			docopy(f, from.Field(i))
			//f.Set(from.Field(i))
		}
	}
}

func docopy(to, from reflect.Value) {
	to = reflect.Indirect(to)
	from = reflect.Indirect(from)
	if to.Type() == from.Type() {
		switch to.Type().Kind() {
		case reflect.Struct:
			for i := 0; i < to.NumField(); i++ {
				docopy(to.Field(i), from.Field(i))
			}
		case reflect.Slice:
			if to.Cap() > from.Len() {
				to.SetLen(from.Len())
			} else {
				to.Set(reflect.MakeSlice(to.Type(), from.Len(), from.Len()))
			}
			for i := 0; i < from.Len(); i++ {
				to.Index(i).Set(from.Index(i))
			}
		default:
			to.Set(from)
		}

	}
}
