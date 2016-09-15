/*

Port of File-RotateLogs from Perl (https://metacpan.org/release/File-RotateLogs)

*/

package rotatelogs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"bitbucket.org/tebeka/strftime"
)

type RotateLogs struct {
	clock        Clock
	curFn        string
	globPattern  string
	linkName     string
	maxAge       time.Duration
	mutex        sync.Mutex
	offset       time.Duration
	outFh        *os.File
	pattern      string
	rotationTime time.Duration
}

type Clock interface {
	Now() time.Time
}
type clockFn func() time.Time

func (c clockFn) Now() time.Time {
	return c()
}

type Option interface {
	Set(*RotateLogs) error
}

type OptionFn func(*RotateLogs) error

func (o OptionFn) Set(rl *RotateLogs) error {
	return o(rl)
}

func WithClock(c Clock) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.clock = c
		return nil
	})
}

func WithLinkName(s string) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.linkName = s
		return nil
	})
}

func WithMaxAge(d time.Duration) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.maxAge = d
		return nil
	})
}

func WithOffset(d time.Duration) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.offset = d
		return nil
	})
}

func WithRotationTime(d time.Duration) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.rotationTime = d
		return nil
	})
}

func New(pattern string, options ...Option) *RotateLogs {
	globPattern := pattern
	for _, re := range patternConversionRegexps {
		globPattern = re.ReplaceAllString(globPattern, "*")
	}

	var rl RotateLogs
	rl.clock = clockFn(time.Now)
	rl.globPattern = globPattern
	rl.pattern = pattern
	rl.rotationTime = 24 * time.Hour
	for _, opt := range options {
		opt.Set(&rl)
	}

	return &rl
}

func (rl *RotateLogs) GenFilename() (string, error) {
	now := rl.clock.Now()
	diff := time.Duration(now.Add(rl.offset).UnixNano()) % rl.rotationTime
	t := now.Add(time.Duration(-1 * diff))
	str, err := strftime.Format(rl.pattern, t)
	if err != nil {
		return "", err
	}
	return str, err
}

func (rl *RotateLogs) Write(p []byte) (n int, err error) {
	// Guard against concurrent writes
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// This filename contains the name of the "NEW" filename
	// to log to, which may be newer than rl.currentFilename

	filename, err := rl.GenFilename()
	if err != nil {
		return 0, err
	}

	var out *os.File
	if filename == rl.curFn { // Match!
		out = rl.outFh // use old one
	}

	var isNew bool

	if out == nil {
		isNew = true

		_, err := os.Stat(filename)
		if err == nil {
			if rl.linkName != "" {
				_, err = os.Lstat(rl.linkName)
				if err == nil {
					isNew = false
				}
			}
		}

		fh, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, fmt.Errorf("error: Failed to open file %s: %s", rl.pattern, err)
		}

		out = fh
		if isNew {
			rl.rotate(filename)
		}
	}

	n, err = out.Write(p)

	if rl.outFh == nil {
		rl.outFh = out
	} else if isNew {
		rl.outFh.Close()
		rl.outFh = out
	}
	rl.curFn = filename

	return n, err
}

func (rl *RotateLogs) CurrentFileName() string {
	return rl.curFn
}

var patternConversionRegexps = []*regexp.Regexp{
	regexp.MustCompile(`%[%+A-Za-z]`),
	regexp.MustCompile(`\*+`),
}

type cleanupGuard struct {
	enable bool
	fn     func()
	mutex  sync.Mutex
}

func (g *cleanupGuard) Enable() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.enable = true
}
func (g *cleanupGuard) Run() {
	g.fn()
}

func (rl *RotateLogs) rotate(filename string) error {
	lockfn := fmt.Sprintf("%s_lock", filename)

	fh, err := os.OpenFile(lockfn, os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		// Can't lock, just return
		return err
	}

	var guard cleanupGuard
	guard.fn = func() {
		fh.Close()
		os.Remove(lockfn)
	}
	defer guard.Run()

	if rl.linkName != "" {
		tmpLinkName := fmt.Sprintf("%s_symlink", filename)
		err = os.Symlink(filename, tmpLinkName)
		if err != nil {
			return err
		}

		err = os.Rename(tmpLinkName, rl.linkName)
		if err != nil {
			return err
		}
	}

	if rl.maxAge <= 0 {
		return errors.New("maxAge not set, not rotating")
	}

	matches, err := filepath.Glob(rl.globPattern)
	if err != nil {
		return err
	}

	cutoff := rl.clock.Now().Add(-1 * rl.maxAge)
	var toUnlink []string
	for _, path := range matches {
		// Ignore lock files
		if strings.HasSuffix(path, "_lock") || strings.HasSuffix(path, "_symlink") {
			continue
		}

		fi, err := os.Stat(path)
		if err != nil {
			continue
		}

		if fi.ModTime().After(cutoff) {
			continue
		}
		toUnlink = append(toUnlink, path)
	}

	if len(toUnlink) <= 0 {
		return errors.New("nothing to unlink")
	}

	guard.Enable()
	go func() {
		// unlink files on a separate goroutine
		for _, path := range toUnlink {
			os.Remove(path)
		}
	}()

	return nil
}

func (rl *RotateLogs) Close() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if rl.outFh == nil {
		return nil
	}

	rl.outFh.Close()
	rl.outFh = nil
	return nil
}
