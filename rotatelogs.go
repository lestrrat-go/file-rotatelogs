/*

Port of File-RotateLogs from Perl (https://metacpan.org/release/File-RotateLogs)

*/

package rotatelogs

import (
	"bitbucket.org/tebeka/strftime"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type RotateLogs struct {
	LogFile      string
	LinkName     string
	RotationTime time.Duration
	MaxAge       time.Duration
	Offset       time.Duration

	curFn          string
	outFh          *os.File
	logfilePattern string
	sem            chan bool
}

/* CurrentTime is only used for testing. Normally it's the time.Now()
 * function
 */
var CurrentTime = time.Now

func NewRotateLogs(logfile string) *RotateLogs {
	return &RotateLogs{
		logfile,
		"",
		86400 * time.Second,
		0,
		0,
		"",
		nil,
		"",
		make(chan bool, 1),
	}
}

func (rl *RotateLogs) GenFilename() (string, error) {
	now := CurrentTime()
	diff := time.Duration(now.Add(rl.Offset).UnixNano()) % rl.RotationTime
	t := now.Add(time.Duration(-1 * diff))
	str, err := strftime.Format(rl.LogFile, t)
	if err != nil {
		return "", err
	}
	return str, err
}

func (rl *RotateLogs) Write(p []byte) (n int, err error) {
	// Guard against concurrent writes
	rl.sem <- true
	defer func() { <-rl.sem }()

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

	isNew := false
	if out == nil {
		isNew = true

		_, err := os.Stat(filename)
		if err == nil {
			if rl.LinkName != "" {
				_, err = os.Lstat(rl.LinkName)
				if err == nil {
					isNew = false
				}
			}
		}

		fh, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, fmt.Errorf("error: Failed to open file %s: %s", rl.LogFile, err)
		}

		out = fh
		if isNew {
			rl.Rotate(filename)
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

func (rl *RotateLogs) LogFilePattern() string {
	if rl.logfilePattern == "" {
		lf := rl.LogFile

		for _, re := range patternConversionRegexps {
			lf = re.ReplaceAllString(lf, "*")
		}
		rl.logfilePattern = lf
	}
	return rl.logfilePattern
}

func (rl *RotateLogs) Rotate(filename string) error {
	lockfn := fmt.Sprintf("%s_lock", filename)
	fh, err := os.OpenFile(lockfn, os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		// Can't lock, just return
		return err
	}

	goneToBackground := false
	guard := func() {
		fh.Close()
		os.Remove(lockfn)
	}
	defer func() {
		if !goneToBackground {
			guard()
		}
	}()

	if rl.LinkName != "" {
		tmpLinkName := fmt.Sprintf("%s_symlink", filename)
		err = os.Symlink(filename, tmpLinkName)
		if err != nil {
			return err
		}

		err = os.Rename(tmpLinkName, rl.LinkName)
		if err != nil {
			return err
		}
	}

	if rl.MaxAge <= 0 {
		return nil
	}

	pattern := rl.LogFilePattern()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	cutoff := CurrentTime().Add(-1 * rl.MaxAge)
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
		return nil
	}

	goneToBackground = true
	go func() {
		defer guard()
		// unlink files on a separate goroutine
		for _, path := range toUnlink {
			os.Remove(path)
		}
	}()

	return nil
}

func (rl *RotateLogs) Close() error {
	if rl.outFh != nil {
		rl.outFh.Close()
		rl.outFh = nil
	}
	return nil
}
