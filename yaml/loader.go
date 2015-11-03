// Package yaml defines a loader for yaml config files
// 	autoconfig.Load(ini.New(filename))
package yaml

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Loader struct {
	filename string
}

// New creates a Loader for INI files
func New(filename string) *Loader {
	return &Loader{filename: filename}
}

// Load loads the config file and unmarshals it to cfg
func (l *Loader) Load(cfg map[string]interface{}) error {
	data, err := ioutil.ReadFile(l.filename)
	if err != nil {
		return err
	}

	tmp := map[string]interface{}{}
	err = yaml.Unmarshal(data, tmp)
	for name, scfg := range cfg {
		if syam, ok := tmp[name]; ok {
			buf, err := yaml.Marshal(syam)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(buf, scfg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
