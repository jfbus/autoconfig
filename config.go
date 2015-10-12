package config

import (
	"encoding/json"
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

type Config struct {
	filename string
	sections map[string]*section
	current  map[string]interface{}
	loader   Loader
}

type UpdatableConfig interface {
	Changed()
}

type Reconfigurable interface {
	Reconfigure(interface{})
}

type Loader interface {
	Load(map[string]interface{}) error
}

var globalConfig = Config{sections: map[string]*section{}, current: map[string]interface{}{}}

func New(l Loader) *Config {
	return &Config{sections: map[string]*section{}, current: map[string]interface{}{}, loader: l}
}

func (c *Config) Load() error {
	return c.load()
}

func Load(l Loader) error {
	globalConfig.loader = l
	return globalConfig.Load()
}

func (c *Config) Reload() error {
	return globalConfig.Load()
}

func Reload() error {
	return globalConfig.Reload()
}

func (c *Config) ReloadOn(signals ...os.Signal) {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, signals...)
		for range ch {
			c.Reload()
		}
	}()
}

func ReloadOn(signals ...os.Signal) {
	globalConfig.ReloadOn(signals...)
}

func (c *Config) Register(name string, s interface{}) bool {
	if uc, ok := s.(UpdatableConfig); ok {
		c.register(name, s, &reconfigurableCfg{uc})
	} else {
		c.register(name, s, nil)
	}
	return true
}

func Register(name string, s interface{}) bool {
	return globalConfig.Register(name, s)
}

func (c *Config) Reconfigure(name string, r Reconfigurable) bool {
	c.register(name, nil, r)
	if cfg, ok := c.Get(name); ok {
		r.Reconfigure(cfg)
	}
	return true
}

func Reconfigure(name string, r Reconfigurable) bool {
	return globalConfig.Reconfigure(name, r)
}

func (c *Config) Get(name string) (interface{}, bool) {
	cfg, ok := c.current[name]
	return cfg, ok
}

func (c *Config) MustGet(name string) interface{} {
	return c.current[name]
}

func Get(name string) (interface{}, bool) {
	return globalConfig.Get(name)
}

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
		addDefaults(c.sections[name].defaults, v)
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

func addDefaults(to, from reflect.Value) {
	to = reflect.Indirect(to)
	from = reflect.Indirect(from)
	for i := 0; i < to.NumField(); i++ {
		f := to.Field(i)
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
		// TODO : case reflect.Slice:
		default:
			to.Set(from)
		}

	}
}
