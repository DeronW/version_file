package version_file

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/go-playground/validator/v10"
)

type fileInfo struct {
	Sha1 string    `json:"sha1" validate:"required"`
	Time time.Time `json:"time" validate:"required"`
}

type versionFile struct {
	Version string `json:"version"`
	Dir     string `json:"dir" validate:"required"`
	Length  int    `json:"length"`
	Current int    `json:"current"`
	Left    int    `json:"left"`
	Right   int    `json:"right"`
	// used for files reuse detect
	Once string `json:"once" validate:"required"`
	// record every version's hash
	Files map[string]fileInfo `json:"files"`
}

var (
	validate = validator.New(validator.WithRequiredStructEnabled())
)

func New(dir string) (versionFile, error) {
	v := versionFile{
		Version: "0.1.0",
		Dir:     dir,
		Length:  10,
		Once:    fmt.Sprintf("%d.%f", time.Now().Unix(), rand.Float64()),
	}
	f := v.file(0)
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return v, v.note()
	}

	bs, err := os.ReadFile(v.file(0))
	if err != nil {
		return v, err
	}

	err = json.Unmarshal(bs, &v)
	return v, err
}

// set history lenght will drop all history
func (v *versionFile) SetLength(n int) error {
	if n < 1 {
		return errors.New("history length should at least 1")
	}
	if n > 10_000 {
		return errors.New("history length is too long")
	}

	bs, err := v.Pick(v.Current)
	if err != nil {
		return err
	}
	err = v.write(0, bs)
	if err != nil {
		return err
	}

	v.Left = 0
	v.Right = 0
	v.Length = n
	return v.note()
}

func (v *versionFile) Back() error {
	if v.Left == 0 {
		return errors.New("no backward steps")
	}
	v.Current = v.left()
	v.Left += 1
	v.Right += 1
	return v.note()
}

func (v *versionFile) Forward() error {
	if v.Right == 0 {
		return errors.New("no forward steps")
	}
	v.Current = v.right()
	v.Left -= 1
	v.Right -= 1
	return v.note()
}

// add a new version of file
func (v *versionFile) Push(bs []byte) error {
	n := v.right()
	err := v.write(n, bs)
	if err != nil {
		return err
	}
	v.Current = n
	v.Right = 0
	if v.Left+v.Length > 1 {
		v.Left -= 1
	}
	v.Files[fmt.Sprintf("%d", n)] = fileInfo{
		Sha1: hash(bs),
		Time: time.Now(),
	}
	return v.note()
}

func (v *versionFile) PushJson(obj any) error {
	err := validate.Struct(obj)
	if err != nil {
		return err
	}
	bs, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return v.Push(bs)
}

// reset to sepecific version
func (v *versionFile) Reset(n int) ([]byte, error) {
	bs, err := v.Pick(n)
	if err != nil {
		return nil, err
	}
	v.Current += n
	v.Left -= n
	v.Right -= n
	err = v.note()
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (v *versionFile) Pick(n int) ([]byte, error) {
	if n < v.Left {
		return nil, errors.New("no such older version")
	}
	if n > v.Right {
		return nil, errors.New("no such newer version")
	}
	return v.read(n)
}

func (v *versionFile) note() error {
	bs, err := json.Marshal(*v)
	if err != nil {
		return err
	}
	return v.write(0, bs)
}

func (v *versionFile) right() int {
	n := v.Current + 1
	if n > v.Length {
		n = 1
	}
	return n
}

func (v *versionFile) left() int {
	n := v.Current - 1
	if n < 1 {
		n = v.Length
	}
	return n
}

func (v *versionFile) file(n int) string {
	return path.Join(v.Dir, fmt.Sprintf("%d.json", n))
}

func (v *versionFile) read(n int) ([]byte, error) {
	return os.ReadFile(v.file(n))
}

func (v *versionFile) write(n int, bs []byte) error {
	return os.WriteFile(v.file(n), bs, 0644)
}

func hash(bs []byte) string {
	h := sha1.New()
	h.Write(bs)
	return fmt.Sprintf("%x", h.Sum(nil))
}
