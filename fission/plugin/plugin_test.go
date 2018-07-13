package plugin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	os.Clearenv()
	testDir := path.Join(os.TempDir(), fmt.Sprintf("fission-test-plugins-%v", time.Now().UnixNano()))
	err := os.MkdirAll(testDir, os.ModePerm)
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(testDir)
	testBinary := path.Join(testDir, "foo")
	md := &Metadata{
		Name:    "foo",
		Version: "1.0.1",
		Usage:   "Usage help",
		Aliases: []string{"bar"},
	}
	jsonMd, err := json.Marshal(md)
	if err != nil {
		t.FailNow()
	}
	err = ioutil.WriteFile(testBinary, []byte(fmt.Sprintf("#!/bin/sh\necho '%v'", string(jsonMd))), os.ModePerm)
	if err != nil {
		t.FailNow()
	}

	err = os.Setenv("PATH", testDir)
	if err != nil {
		t.FailNow()
	}
	Prefix = ""

	found, err := Find(md.Name)
	os.RemoveAll(testDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, found)
	assert.Equal(t, md.Name, found.Name)
	assert.Equal(t, path.Join(testDir, md.Name), found.Path)
	assert.Equal(t, md.Aliases, found.Aliases)
	assert.Equal(t, md.Usage, found.Usage)
	assert.Equal(t, md.Version, found.Version)
}

func TestExec(t *testing.T) {
	os.Clearenv()
	testDir := path.Join(os.TempDir(), fmt.Sprintf("fission-test-plugins-%v", time.Now().UnixNano()))
	err := os.MkdirAll(testDir, os.ModePerm)
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(testDir)
	testBinary := path.Join(testDir, "foo")
	md := &Metadata{
		Name:    "foo",
		Version: "1.0.1",
		Usage:   "Usage help",
		Aliases: []string{"bar"},
		Path:    path.Join(testBinary),
	}
	jsonMd, err := json.Marshal(md)
	if err != nil {
		t.FailNow()
	}
	err = ioutil.WriteFile(testBinary, []byte(fmt.Sprintf("#!/bin/sh\necho '%v'", string(jsonMd))), os.ModePerm)
	if err != nil {
		t.FailNow()
	}
	err = os.Setenv("PATH", testDir)
	if err != nil {
		t.FailNow()
	}
	Prefix = ""
	err = Exec(md, nil)
	os.RemoveAll(testDir)
	assert.NoError(t, err)
}
