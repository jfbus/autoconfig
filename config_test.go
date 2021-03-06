package autoconfig

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/jfbus/autoconfig/ini"
	"github.com/jfbus/autoconfig/yaml"
)

type testCfg struct {
	Key     string `ini:"key" yaml:"key"`
	None    string `ini:"none" yaml:"none"`
	changed int
}

func (t *testCfg) Changed() {
	t.changed++
}

func (t *testCfg) changeCount() int {
	return t.changed
}

type Deeper struct {
	Key string `ini:"key" yaml:"key"`
}

type testDeepCfg struct {
	Deeper  Deeper `yaml:"deeper"`
	None    string `ini:"none" yaml:"none"`
	changed int
}

func (t *testDeepCfg) Changed() {
	t.changed++
}

func (t *testDeepCfg) changeCount() int {
	return t.changed
}

type testSliceCfg struct {
	Key     []string `yaml:"key"`
	changed int
}

func (t *testSliceCfg) Changed() {
	t.changed++
}

func (t *testSliceCfg) changeCount() int {
	return t.changed
}

type testCfgMap map[string]int

func (t *testCfgMap) Changed() {
	m := *t
	if _, ok := m["changed"]; !ok {
		m["changed"] = 0
	}
	m["changed"]++
}

func (t *testCfgMap) changeCount() int {
	m := *t
	if c, ok := m["changed"]; ok {
		return c
	} else {
		return 0
	}
}

type testClass struct {
	cfg     *testCfg
	changed int
}

func (t *testClass) Reconfigure(c interface{}) {
	t.cfg, _ = c.(*testCfg)

	t.changed++
}

func (t *testClass) changeCount() int {
	return t.changed
}

type changeCounter interface {
	changeCount() int
}

type testLoaderInterface interface {
	loader(string) (Loader, error)
	update(string) error
	clean()
}

type testLoader struct {
	f *os.File
}

func (l *testLoader) write(raw string) error {
	var err error
	l.f, err = ioutil.TempFile("/tmp/", "autoconfig_test_")
	if err != nil {
		return err
	}
	_, err = l.f.WriteString(raw)
	if err != nil {
		return err
	}
	return l.f.Sync()
}

func (l *testLoader) update(raw string) error {
	err := l.f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = l.f.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = l.f.WriteString(raw)
	if err != nil {
		return err
	}
	return l.f.Close()
}

func (l *testLoader) clean() {
	os.Remove(l.f.Name())
}

type iniLoader struct {
	testLoader
}

func (l *iniLoader) loader(raw string) (Loader, error) {
	err := l.write(raw)
	if err != nil {
		return nil, err
	}
	return ini.New(l.f.Name()), nil
}

type yamlLoader struct {
	testLoader
}

func (l *yamlLoader) loader(raw string) (Loader, error) {
	err := l.write(raw)
	if err != nil {
		return nil, err
	}
	return yaml.New(l.f.Name()), nil
}

type testCase struct {
	name        string
	raw         string
	rawUpdated  string
	loader      testLoaderInterface
	defaults    func() changeCounter
	afterLoad   changeCounter
	afterUpdate changeCounter
}

var (
	iniRaw = `
[section]
key=foo
`
	iniRawUpdated = `
[section]
key=bar
`

	testCases = []testCase{
		testCase{
			name:        "ini",
			raw:         iniRaw,
			rawUpdated:  iniRawUpdated,
			loader:      &iniLoader{},
			defaults:    func() changeCounter { return &testCfg{None: "foobar"} },
			afterLoad:   &testCfg{Key: "foo", None: "foobar", changed: 1},
			afterUpdate: &testCfg{Key: "bar", None: "foobar", changed: 2},
		},
		testCase{
			name: "yaml flat",
			raw: `section:
  key: foo
`,
			rawUpdated: `section:
  key: bar
`,
			loader:      &yamlLoader{},
			defaults:    func() changeCounter { return &testCfg{None: "foobar"} },
			afterLoad:   &testCfg{Key: "foo", None: "foobar", changed: 1},
			afterUpdate: &testCfg{Key: "bar", None: "foobar", changed: 2},
		},
		testCase{
			name: "yaml deep",
			raw: `section:
  deeper:
    key: foo
`,
			rawUpdated: `section:
  deeper:
    key: bar
`,
			loader:      &yamlLoader{},
			defaults:    func() changeCounter { return &testDeepCfg{None: "foobar"} },
			afterLoad:   &testDeepCfg{Deeper: Deeper{Key: "foo"}, None: "foobar", changed: 1},
			afterUpdate: &testDeepCfg{Deeper: Deeper{Key: "bar"}, None: "foobar", changed: 2},
		},
		testCase{
			name: "yaml slice no defaults",
			raw: `section:
  key: ["foo"]
`,
			rawUpdated: `section:
  key: ["foo", "bar"]
`,
			loader:      &yamlLoader{},
			defaults:    func() changeCounter { return &testSliceCfg{} },
			afterLoad:   &testSliceCfg{Key: []string{"foo"}, changed: 1},
			afterUpdate: &testSliceCfg{Key: []string{"foo", "bar"}, changed: 2},
		},
		testCase{
			name: "yaml slice defaults",
			raw: `section:
`,
			rawUpdated: `section:
  key: ["foo", "bar"]
`,
			loader:      &yamlLoader{},
			defaults:    func() changeCounter { return &testSliceCfg{Key: []string{"foo"}} },
			afterLoad:   &testSliceCfg{Key: []string{"foo"}, changed: 1},
			afterUpdate: &testSliceCfg{Key: []string{"foo", "bar"}, changed: 2},
		},
		testCase{
			name: "yaml slice none",
			raw: `section:
`,
			rawUpdated: `section:
  key: ["foo", "bar"]
`,
			loader:      &yamlLoader{},
			defaults:    func() changeCounter { return &testSliceCfg{} },
			afterLoad:   &testSliceCfg{changed: 1},
			afterUpdate: &testSliceCfg{Key: []string{"foo", "bar"}, changed: 2},
		},
		testCase{
			name: "yaml map",
			raw: `section:
  key: 21
`,
			rawUpdated: `section:
  key: 42
`,
			loader:      &yamlLoader{},
			defaults:    func() changeCounter { return &testCfgMap{} },
			afterLoad:   &testCfgMap{"key": 21, "changed": 1},
			afterUpdate: &testCfgMap{"key": 42, "changed": 2},
		},
	}
)

func TestReloadCfg(t *testing.T) {
	for _, tc := range testCases {
		l, err := tc.loader.loader(tc.raw)
		if err != nil {
			t.Fatal("Unable to create config temp file")
		}
		cfg := New(l)
		scfg := tc.defaults()
		cfg.Register("section", scfg)
		err = cfg.Load()
		if err != nil {
			t.Errorf("When loading %s conf, Load() returned %s", tc.name, err)
		}
		if !reflect.DeepEqual(scfg, tc.afterLoad) {
			t.Errorf("When loading %s conf, expected <%#v>, got <%#v>", tc.name, tc.afterLoad, scfg)
		}
		tc.loader.update(tc.rawUpdated)
		err = cfg.Reload()
		if err != nil {
			t.Errorf("When loading %s conf, Reload() returned %s", tc.name, err)
		}
		if !reflect.DeepEqual(scfg, tc.afterUpdate) {
			t.Errorf("When reloading %s conf, expected <%#v>, got <%#v>", tc.name, tc.afterUpdate, scfg)
		}
		tc.loader.clean()
	}
}

func TestAfterLoadCfg(t *testing.T) {
	for _, tc := range testCases {
		l, err := tc.loader.loader(tc.raw)
		if err != nil {
			t.Fatal("Unable to create config temp file")
		}
		cfg := New(l)
		scfg := tc.defaults()
		err = cfg.Load()
		if err != nil {
			t.Errorf("When loading %s conf, Load() returned %s", tc.name, err)
		}
		cfg.Register("section", scfg)
		if !reflect.DeepEqual(scfg, tc.afterLoad) {
			t.Errorf("When loading %s conf, expected <%#v>, got <%#v>", tc.name, tc.afterLoad, scfg)
		}
		tc.loader.clean()
	}
}

func TestReloadInstance(t *testing.T) {
	tc := testCases[0]
	l, err := tc.loader.loader(tc.raw)
	if err != nil {
		t.Fatal("Unable to create config temp file")
	}
	cfg := New(l)
	cfg.Register("section", tc.defaults())
	i := &testClass{}
	cfg.Reconfigure("section", i)
	err = cfg.Load()
	if err != nil {
		t.Errorf("When loading %s conf, Load() returned %s", tc.name, err)
	}
	if !reflect.DeepEqual(i.cfg, tc.afterLoad) {
		t.Errorf("When loading %s conf, expected <%#v>, got <%#v>", tc.name, tc.afterLoad, i.cfg)
	}
	tc.loader.update(tc.rawUpdated)
	err = cfg.Reload()
	if err != nil {
		t.Errorf("When loading %s conf, Reload() returned %s", tc.name, err)
	}
	if !reflect.DeepEqual(i.cfg, tc.afterUpdate) {
		t.Errorf("When reloading %s conf, expected <%#v>, got <%#v>", tc.name, tc.afterUpdate, i.cfg)
	}
	tc.loader.clean()
}

func TestAfterLoadInstance(t *testing.T) {
	tc := testCases[0]
	l, err := tc.loader.loader(tc.raw)
	if err != nil {
		t.Fatal("Unable to create config temp file")
	}
	cfg := New(l)
	err = cfg.Load()
	cfg.Register("section", tc.defaults())
	i := &testClass{}
	cfg.Reconfigure("section", i)
	if err != nil {
		t.Errorf("When loading %s conf, Load() returned %s", tc.name, err)
	}
	if !reflect.DeepEqual(i.cfg, tc.afterLoad) {
		t.Errorf("When loading %s conf, expected <%#v>, got <%#v>", tc.name, tc.afterLoad, i.cfg)
	}
	tc.loader.clean()
}

func TestNoLoader(t *testing.T) {
	err := Load(nil)
	if err != ErrNoLoader {
		t.Errorf("When no loader is defined, Load should return <%s>, got <%s>", ErrNoLoader, err)
	}
}
