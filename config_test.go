package autoconfig

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jfbus/autoconfig/ini"
)

var (
	iniCfgBefore = `
[section]
key=foo
`
	iniCfgAfter = `
[section]
key=bar
`
)

type testCfg struct {
	Key     string `ini:"key"`
	None    string `ini:"none"`
	changed int
}

func (t *testCfg) Changed() {
	t.changed++
}

type testClass struct {
	cfg     *testCfg
	changed int
}

func (t *testClass) Reconfigure(c interface{}) {
	t.cfg, _ = c.(*testCfg)

	t.changed++
}

func TestNoLoader(t *testing.T) {
	err := Load(nil)
	if err != ErrNoLoader {
		t.Errorf("When no loader is defined, Load should return <%s>, got <%s>", ErrNoLoader, err)
	}
}

func TestLoadCfg(t *testing.T) {
	f, err := ioutil.TempFile("/tmp/", "autoconfig_test_")
	if err != nil {
		t.Fatal("Unable to create config temp file")
	}
	defer os.Remove(f.Name())
	f.WriteString(iniCfgBefore)
	f.Close()
	cfg := New(ini.New(f.Name()))
	scfg := &testCfg{None: "foobar"}
	cfg.Register("section", scfg)
	cfg.Load()
	if scfg.None != "foobar" {
		t.Errorf("When loading conf, defaults should be present - expected <foobar>, got <%s>", scfg.None)
	}
	if scfg.Key != "foo" {
		t.Errorf("When loading conf, values from file should be present - expected <foo>, got <%s>", scfg.Key)
	}
}

func TestReloadCfg(t *testing.T) {
	f, err := ioutil.TempFile("/tmp/", "autoconfig_test_")
	if err != nil {
		t.Fatal("Unable to create config temp file")
	}
	defer os.Remove(f.Name())
	f.WriteString(iniCfgBefore)
	f.Sync()
	cfg := New(ini.New(f.Name()))
	scfg := &testCfg{None: "foobar"}
	cfg.Register("section", scfg)
	cfg.Load()
	if scfg.None != "foobar" {
		t.Errorf("When loading conf, defaults should be present - expected <foobar>, got <%s>", scfg.None)
	}
	if scfg.Key != "foo" {
		t.Errorf("When loading conf, values from file should be present - expected <foo>, got <%s>", scfg.Key)
	}
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(iniCfgAfter)
	f.Close()
	c := scfg.changed
	cfg.Reload()
	if scfg.changed == c {
		t.Errorf("When reloading conf, Changed should have been called")
	}
	if scfg.Key != "bar" {
		t.Errorf("When reloading conf, new values from file should be present - expected <bar>, got <%s>", scfg.Key)
	}
}

func TestReloadInstance(t *testing.T) {
	f, err := ioutil.TempFile("/tmp/", "autoconfig_test_")
	if err != nil {
		t.Fatal("Unable to create config temp file")
	}
	defer os.Remove(f.Name())
	f.WriteString(iniCfgBefore)
	f.Sync()
	cfg := New(ini.New(f.Name()))
	scfg := &testCfg{None: "foobar"}
	cfg.Register("section", scfg)
	i := &testClass{}
	cfg.Reconfigure("section", i)
	cfg.Load()
	if i.cfg.Key != "foo" {
		t.Errorf("When loading conf, values from file should be present - expected <foo>, got <%s>", scfg.Key)
	}
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(iniCfgAfter)
	f.Close()
	c := i.cfg.changed
	cfg.Reload()
	if i.cfg.changed == c {
		t.Errorf("When reloading conf, Changed should have been called")
	}
	if i.cfg.Key != "bar" {
		t.Errorf("When reloading conf, new values from file should be present - expected <bar>, got <%s>", scfg.Key)
	}
}
