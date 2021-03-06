package web

import (
	"crypto/hmac"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/zond/hackyhack/logging"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/resource"
	"github.com/zond/hackyhack/server/router"
	"github.com/zond/hackyhack/server/router/validator"
	"github.com/zond/hackyhack/server/user"
)

var editorTmpl *template.Template

const (
	realm = "hackyhack"
)

func init() {
	editorTmpl = template.Must(template.ParseFiles(filepath.Join(
		os.Getenv("GOPATH"),
		"src",
		"github.com",
		"zond",
		"hackyhack",
		"server",
		"web",
		"static",
		"editor.html",
	)))
	rand.Seed(time.Now().UnixNano())
}

type webErr struct {
	status int
	body   string
}

func (w webErr) Error() string {
	return fmt.Sprintf("%v: %v", w.body, w.status)
}

type context struct {
	user *user.User
	req  *http.Request
	resp http.ResponseWriter
	vars map[string]string
}

type Web struct {
	persister  *persist.Persister
	muxRouter  *mux.Router
	hackRouter *router.Router
}

type memRespWriter struct {
	http.ResponseWriter
	status int
}

func (m *memRespWriter) Header() http.Header {
	return m.ResponseWriter.Header()
}

func (m *memRespWriter) Write(b []byte) (int, error) {
	return m.ResponseWriter.Write(b)
}

func (m *memRespWriter) WriteHeader(i int) {
	m.status = i
	m.ResponseWriter.WriteHeader(i)
}

func (web *Web) log(f func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memW := &memRespWriter{
			ResponseWriter: w,
			status:         200,
		}
		w = memW
		defer logging.Outputter(log.Printf).Tracef(func(first bool) string {
			if first {
				return fmt.Sprintf("WEB\t%v\t%v", r.Method, r.URL.String())
			} else {
				return fmt.Sprintf("WEB\t%v\t%v\t%v", r.Method, r.URL.String(), memW.status)
			}
		})()
		f(w, r)
	}
}

func New(p *persist.Persister, r *router.Router) *Web {
	web := &Web{
		persister:  p,
		muxRouter:  mux.NewRouter(),
		hackRouter: r,
	}
	web.muxRouter.Path("/favicon.ico").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not found", 404) })
	web.muxRouter.PathPrefix("/static").HandlerFunc(web.log(http.StripPrefix("/static", http.FileServer(http.Dir(filepath.Join(
		os.Getenv("GOPATH"),
		"src",
		"github.com",
		"zond",
		"hackyhack",
		"server",
		"web",
		"static",
	)))).ServeHTTP))
	web.muxRouter.Path("/edit/{resource}").Methods("GET").HandlerFunc(web.authenticated(web.editor))
	web.muxRouter.Path("/{resource}").Methods("GET").HandlerFunc(web.authenticated(web.getResource))
	web.muxRouter.Path("/{resource}").Methods("PUT").HandlerFunc(web.authenticated(web.putResource))
	return web
}

func (web *Web) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	web.muxRouter.ServeHTTP(w, r)
}

func (web *Web) authenticated(f func(*context) error) http.HandlerFunc {
	return web.log(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			newURL := r.URL
			newURL.Scheme = "https"
			http.Redirect(w, r, newURL.String(), 301)
			return
		}

		username, passwd, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", realm))
			http.Error(w, "Unauthenticated", 401)
			return
		}

		users := []user.User{}
		if err := web.persister.Find(persist.NewF(user.User{
			Username: username,
		}).Add("Username"), &users); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if len(users) == 0 {
			http.Error(w, "Unauthenticated", 401)
			return
		}

		var user *user.User
		for _, found := range users {
			if hmac.Equal([]byte(found.Password), []byte(passwd)) {
				user = &found
				break
			}
		}

		if user == nil {
			http.Error(w, "Unauthenticated", 401)
			return
		}

		if err := f(&context{
			user: user,
			req:  r,
			resp: w,
			vars: mux.Vars(r),
		}); err != nil {
			if werr, ok := err.(webErr); ok {
				http.Error(w, werr.body, werr.status)
			} else {
				http.Error(w, err.Error(), 500)
			}
			return
		}
	})
}

type editorContext struct {
	Resource *resource.Resource
	User     *user.User
}

func (web *Web) editor(c *context) error {
	res := &resource.Resource{}
	if err := web.persister.Get(c.vars["resource"], res); err == persist.ErrNotFound {
		return webErr{status: 404, body: fmt.Sprintf("No such resource")}
	} else if err != nil {
		return err
	}
	return editorTmpl.Execute(c.resp, editorContext{
		Resource: res,
		User:     c.user,
	})
}

func (web *Web) getResource(c *context) error {
	res := &resource.Resource{}
	if err := web.persister.Get(c.vars["resource"], res); err != nil {
		return err
	}
	_, err := io.WriteString(c.resp, res.Code)
	return err
}

func (web *Web) putResource(c *context) error {
	res := &resource.Resource{}
	if err := web.persister.Get(c.vars["resource"], res); err == persist.ErrNotFound {
		return webErr{status: 404, body: err.Error()}
	} else if err != nil {
		return err
	}

	if res.Owner != c.user.Resource {
		return webErr{status: 403, body: "Not owner"}
	}

	tmpFileBase := filepath.Join(os.TempDir(), fmt.Sprintf("%x%x", rand.Int63(), rand.Int63()))
	tmpFileName := fmt.Sprintf("%v.go", tmpFileBase)
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	if err := func() error {
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, c.req.Body); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		return err
	}
	defer os.Remove(tmpFileName)

	output, err := exec.Command("goimports", "-w", tmpFileName).CombinedOutput()
	if err != nil {
		return err
	}
	if len(output) > 0 {
		return webErr{status: 400, body: string(output)}
	}

	body, err := ioutil.ReadFile(tmpFileName)
	if err != nil {
		return err
	}

	if err := validator.Validate(string(body)); err != nil {
		return webErr{status: 400, body: err.Error()}
	}

	output, err = exec.Command("go", "build", "-o", tmpFileBase, tmpFileName).CombinedOutput()
	if err != nil {
		return err
	}
	if len(output) > 0 {
		return webErr{status: 400, body: string(output)}
	}

	if err := web.persister.Transact(func(p *persist.Persister) error {
		if err := p.Get(res.Id, res); err != nil {
			return err
		}
		res.Code = string(body)
		return p.Put(res.Id, res)
	}); err != nil {
		return err
	}

	if err := web.hackRouter.Restart(res.Id); err != nil {
		return err
	}

	return nil
}
