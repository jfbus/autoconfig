// Package ini defines a loader for ini config files
// 	autoconfig.Load(ini.New(filename))
package ini

import "gopkg.in/ini.v1"

type Loader struct {
	filename string
}

// New creates a loader for ini files
func New(filename string) *Loader {
	return &Loader{filename: filename}
}

func (l *Loader) Load(cfg map[string]interface{}) error {
	f, err := ini.Load(l.filename)
	if err != nil {
		return err
	}
	for name, sec := range cfg {
		s := f.Section(name)
		if s == nil {
			// TODO: raise an error ?
			continue
		}
		err = s.MapTo(sec)
		if err != nil {
			return err
		}
	}
	return nil
}
