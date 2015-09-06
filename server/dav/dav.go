package dav

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/resource"
	"github.com/zond/hackyhack/server/router"
	"github.com/zond/hackyhack/server/user"
	"golang.org/x/net/webdav"
)

type fileInfo struct {
	file *file
}

type file struct {
	fs         *DAVFS
	name       string
	resourceId string
	resource   *resource.Resource
	offset     int
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return int64(len([]byte(f.resource.Code)))
}

func (f *file) Mode() os.FileMode {
	return 0644
}

func (f *file) ModTime() time.Time {
	return f.resource.UpdatedAt
}

func (f *file) IsDir() bool {
	return false
}

func (f *file) Sys() interface{} {
	return nil
}

func (f *file) getFileInfo() (os.FileInfo, error) {
	_, err := f.getResource()
	if err != nil {
		return nil, err
	}
	if f.name == "" {
		shortDesc, merr := util.GetShortDesc(f.fs, f.resourceId)
		if merr != nil {
			return nil, fmt.Errorf("%v: %v", merr.Message, merr.Code)
		}
		f.name = shortDesc
	}
	return f, nil
}

func (f *file) Close() error {
	return nil
}

func (f *file) getResource() (*resource.Resource, error) {
	if f.resource == nil {
		f.resource = &resource.Resource{}
		if err := f.fs.persister.Get(f.resourceId, f.resource); err != nil {
			return nil, err
		}
	}
	return f.resource, nil
}

func (f *file) putResource() error {
	return f.fs.persister.Put(f.resourceId, f.resource)
}

func (f *file) Read(b []byte) (int, error) {
	res, err := f.getResource()
	if err != nil {
		return 0, err
	}
	i := copy(b, []byte(res.Code)[f.offset:])
	f.offset += i
	return i, nil
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("Not a directory")
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.offset = int(offset)
	case 1:
		f.offset += int(offset)
	case 2:
		res, err := f.getResource()
		if err != nil {
			return 0, err
		}
		f.offset = len([]byte(res.Code)) + int(offset)
	}
	return int64(f.offset), nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return f.getFileInfo()
}

func (f *file) Write(b []byte) (int, error) {
	res, err := f.getResource()
	if err != nil {
		return 0, err
	}
	curr := []byte(res.Code)
	i := copy(curr[f.offset:], b)
	if i < len(b) {
		curr = append(curr, b[i:]...)
	}
	f.offset += len(b)
	res.Code = string(curr)
	return len(b), f.putResource()
}

type rootFile struct {
	fs *DAVFS
}

func (f *rootFile) Name() string {
	log.Printf("root name")
	return "/"
}

func (f *rootFile) Size() int64 {
	log.Printf("root size")
	return 0
}

func (f *rootFile) Mode() os.FileMode {
	log.Printf("root mode")
	return 0644 | os.ModeDir
}

func (f *rootFile) ModTime() time.Time {
	log.Printf("root modtime")
	return time.Time{}
}

func (f *rootFile) IsDir() bool {
	log.Printf("root isdir")
	return true
}

func (f *rootFile) Sys() interface{} {
	log.Printf("root sys")
	return nil
}

func (f *rootFile) Readdir(count int) ([]os.FileInfo, error) {
	log.Printf("root readdir")
	shortDescMap, err := util.GetShortDescMap(f.fs, f.fs.user.Resource)
	if err != nil {
		return nil, err.ToErr()
	}

	result := []os.FileInfo{}
	for resource, desc := range shortDescMap {
		file := &file{
			fs:         f.fs,
			name:       desc,
			resourceId: resource,
		}
		info, err := file.getFileInfo()
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}

	return result, nil
}

func (f *rootFile) Seek(offset int64, whence int) (int64, error) {
	log.Printf("root seek (%v, %v)", offset, whence)
	return offset, nil
}

func (f *rootFile) Stat() (os.FileInfo, error) {
	log.Printf("root stat")
	return f, nil
}

func (f *rootFile) Read(b []byte) (int, error) {
	log.Printf("root read (%v)", len(b))
	return 0, io.EOF
}

func (f *rootFile) Write(b []byte) (int, error) {
	log.Printf("root write")
	return 0, fmt.Errorf("Can't write to root")
}

func (f *rootFile) Close() error {
	log.Printf("root close")
	return nil
}

type DAVFS struct {
	persister *persist.Persister
	router    *router.Router
	user      *user.User
	root      *rootFile
}

func (d *DAVFS) GetResource() string {
	return d.user.Resource
}

func (d *DAVFS) Call(resourceId, method string, params, results interface{}) *messages.Error {
	m, err := d.router.MCP(resourceId)
	if err != nil {
		return &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeProxyFailed,
		}
	}

	err = m.Call(d.user.Resource, resourceId, method, params, results)
	if err != nil {
		return &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeProxyFailed,
		}
	}

	return nil
}

func New(p *persist.Persister, r *router.Router, u *user.User) *DAVFS {
	fs := &DAVFS{
		persister: p,
		user:      u,
		router:    r,
	}
	fs.root = &rootFile{
		fs: fs,
	}
	return fs
}

func (d *DAVFS) Mkdir(name string, perm os.FileMode) error {
	log.Printf("fs mkdir")
	return nil
}

func (d *DAVFS) OpenFile(name string, flag int, perm os.FileMode) (webdav.File, error) {
	log.Printf("fs openfile")
	return d.root, nil
}

func (d *DAVFS) RemoveAll(name string) error {
	log.Printf("fs removeall")
	return nil
}

func (d *DAVFS) Rename(oldName, newName string) error {
	log.Printf("fs rename")
	return nil
}

func (d *DAVFS) Stat(name string) (os.FileInfo, error) {
	log.Printf("fs stat")
	return nil, nil
}
