package rotatelogs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenFilename(t *testing.T) {
	// Mock time
	ts := []time.Time{
		time.Time{},
		(time.Time{}).Add(24 * time.Hour),
	}

	var old = CurrentTime
	defer func() { CurrentTime = old }()
	for _, xt := range ts {
		CurrentTime = func() time.Time { return xt }
		rl := NewRotateLogs("/path/to/%Y/%m/%d")
		defer rl.Close()

		fn, err := rl.GenFilename()
		if err != nil {
			t.Errorf("Failed to generate filename: %s", err)
		}

		expected := fmt.Sprintf("/path/to/%04d/%02d/%02d",
			xt.Year(),
			xt.Month(),
			xt.Day(),
		)

		if fn != expected {
			t.Errorf("Failed to match fn (%s)", fn)
		}
		t.Logf("fn = %s", fn)
	}
}

func TestLogFilePattern(t *testing.T) {
	rl := NewRotateLogs("/path/to/%Y/%m/%d")
	defer rl.Close()
	pattern := rl.LogFilePattern()
	if pattern != "/path/to/*/*/*" {
		t.Errorf("Failed to match pattern (%s)", pattern)
	}
}

func TestLogRotate(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-test")
	if err != nil {
		t.Errorf("Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(dir)

	// Change current time, so we can safely purge old logs
	old := CurrentTime
	dummyTime := time.Now().Add(-7 * 86400 * time.Second)
	dummyTime = dummyTime.Add(time.Duration(-1 * dummyTime.Nanosecond()))
	defer func() { CurrentTime = old }()
	CurrentTime = func() time.Time { return dummyTime }

	rl := NewRotateLogs(filepath.Join(dir, "log%Y%m%d%H%M%S"))
	defer rl.Close()

	rl.MaxAge = 86400 * time.Second
	rl.LinkName = filepath.Join(dir, "log")

	str := "Hello, World"
	n, err := rl.Write([]byte(str))
	if n != len(str) {
		t.Errorf("Could not write %d bytes (wrote %d bytes)", len(str), n)
	}

	if err != nil {
		t.Errorf("Failed to Write() to log: %s", err)
	}

	fn := rl.CurrentFileName()
	if fn == "" {
		t.Errorf("Could not get filename %s", fn)
	}

	content, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Errorf("Failed to read file %s: %s", fn, err)
	}

	if string(content) != str {
		t.Errorf(`File content does not match (was "%s")`, content)
	}

	err = os.Chtimes(fn, dummyTime, dummyTime)
	if err != nil {
		t.Errorf("Failed to change access/modification times for %s: %s", fn, err)
	}

	fi, err := os.Stat(fn)
	if err != nil {
		t.Errorf("Failed to stat %s: %s", fn, err)
	}

	if !fi.ModTime().Equal(dummyTime) {
		t.Errorf("Failed to chtime for %s (expected %s, got %s)", fn, fi.ModTime(), dummyTime)
	}

	CurrentTime = old

	// This next Write() should trigger Rotate()
	rl.Write([]byte(str))
	newfn := rl.CurrentFileName()
	if newfn == fn {
		t.Errorf(`New file name and old file name should not match ("%s" != "%s")`, fn, newfn)
	}

	content, err = ioutil.ReadFile(newfn)
	if err != nil {
		t.Errorf("Failed to read file %s: %s", newfn, err)
	}

	if string(content) != str {
		t.Errorf(`File content does not match (was "%s")`, content)
	}

	time.Sleep(1 * time.Second)

	// fn was declared above, before mocking CurrentTime
	// Old files should have been unlinked
	_, err = os.Stat(fn)
	if err == nil {
		t.Errorf("Stat succeeded (should have failed) %s: %s", fn, err)
	}

	linkDest, err := os.Readlink(rl.LinkName)
	if err != nil {
		t.Errorf("Failed to readlink %s: %s", rl.LinkName, err)
	}

	if linkDest != newfn {
		t.Errorf(`Symlink destination does not match expected filename ("%s" != "%s")`, newfn, linkDest)
	}
}

func TestLogSetOutput(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-test")
	if err != nil {
		t.Errorf("Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(dir)

	rl := NewRotateLogs(filepath.Join(dir, "log%Y%m%d%H%M%S"))
	defer rl.Close()

	log.SetOutput(rl)
	defer log.SetOutput(os.Stderr)

	str := "Hello, World"
	log.Print(str)

	fn := rl.CurrentFileName()
	if fn == "" {
		t.Errorf("Could not get filename %s", fn)
	}

	content, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Errorf("Failed to read file %s: %s", fn, err)
	}

	if !strings.Contains(string(content), str) {
		t.Errorf(`File content does not contain "%s" (was "%s")`, str, content)
	}
}
