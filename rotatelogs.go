/*

Port of File-RotateLogs from Perl (https://metacpan.org/release/File-RotateLogs)

*/

package rotatelogs

import(
  "errors"
  "fmt"
  "os"
  "path/filepath"
  "regexp"
  "strings"
  "time"
  "bitbucket.org/tebeka/strftime"
)

type RotateLogs struct {
  LogFile string
  LinkName  string
  RotationTime  time.Duration
  MaxAge        time.Duration
  Offset        time.Duration

  curFn string
  outFh *os.File
  logfilePattern string
  sem chan bool
}

/* CurrentTime is only used for testing. Normally it's the time.Now()
 * function
 */
var CurrentTime = time.Now

func NewRotateLogs(logfile string) (*RotateLogs) {
  return &RotateLogs {
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

func (self *RotateLogs) GenFilename() (string, error) {
  now := CurrentTime()
  diff := time.Duration(now.Add(self.Offset).UnixNano()) % self.RotationTime
  t   := now.Add(time.Duration(-1 * diff))
  str, err :=  strftime.Format(self.LogFile, t)
  if err != nil {
    return "", err
  }
  return str, err
}

func (self *RotateLogs) Write(p []byte) (n int, err error) {
  // Guard against concurrent writes
  self.sem <-true
  defer func() { <-self.sem }()

  // This filename contains the name of the "NEW" filename
  // to log to, which may be newer than self.currentFilename

  filename, err := self.GenFilename()
  if err != nil {
    return 0, err
  }

  var out *os.File
  if filename == self.curFn { // Match!
    out = self.outFh // use old one
  }

  isNew := false
  if out == nil {
    isNew = true

    _, err := os.Stat(filename)
    if err == nil {
      if self.LinkName != "" {
        _, err = os.Lstat(self.LinkName)
        if err == nil {
          isNew = false
        }
      }
    }

    fh, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
      return 0, errors.New(
        fmt.Sprintf("Failed to open file %s: %s", self.LogFile, err),
      )
    }

    out = fh
    if isNew {
      self.Rotate(filename)
    }
  }

  n, err = out.Write(p)

  if isNew && self.outFh != nil {
    self.outFh.Close()
    self.outFh = out
  }
  self.curFn = filename

  return n, err
}

func (self *RotateLogs) CurrentFileName() string {
  return self.curFn
}

var patternConversionRegexps = []*regexp.Regexp {
  regexp.MustCompile(`%[%+A-Za-z]`),
  regexp.MustCompile(`\*+`),
}
func (self *RotateLogs) LogFilePattern() string {
  if self.logfilePattern == "" {
    lf := self.LogFile

    for _, re := range patternConversionRegexps {
      lf = re.ReplaceAllString(lf, "*")
    }
    self.logfilePattern = lf
  }
  return self.logfilePattern
}

func (self *RotateLogs) Rotate(filename string) (error) {
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
    if ! goneToBackground {
      guard()
    }
  }()

  if self.LinkName != "" {
    tmpLinkName := fmt.Sprintf("%s_symlink", filename)
    err = os.Symlink(filename, tmpLinkName)
    if err != nil {
      return err
    }

    err = os.Rename(tmpLinkName, self.LinkName)
    if err != nil {
      return err
    }
  }

  if self.MaxAge <= 0 {
    return nil
  }

  pattern := self.LogFilePattern()
  matches, err := filepath.Glob(pattern)
  if err != nil {
    return err
  }

  cutoff := CurrentTime().Add(-1 * self.MaxAge)
  var to_unlink []string
  for _, path := range matches {
    // Ignore lock files
    if strings.HasSuffix(path, "_lock") || strings.HasSuffix(path, "_symlink"){
      continue
    }

    fi, err := os.Stat(path)
    if err != nil {
      continue
    }

    if fi.ModTime().After(cutoff) {
      continue
    }
    to_unlink = append(to_unlink, path)
  }

  if len(to_unlink) <= 0 {
    return nil
  }

  goneToBackground = true
  go func () {
    defer guard()
    // unlink files on a separate goroutine
    for _, path := range to_unlink {
      os.Remove(path)
    }
  }()

  return nil
}