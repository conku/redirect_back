package redirect_back

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/conku/middlewares"
	"github.com/conku/qor/utils"
	"github.com/conku/session"
	"github.com/conku/session/manager"
)

var returnToKey utils.ContextKey = "redirect_back_return_to"

// Config redirect back config
type Config struct {
	SessionManager    session.ManagerInterface
	FallbackPath      string
	IgnoredPaths      []string
	IgnoredPrefixes   []string
	AllowedExtensions []string
	IgnoreFunc        func(*http.Request) bool
}

// New initialize redirect back instance
func New(config *Config) *RedirectBack {
	if config.SessionManager == nil {
		config.SessionManager = manager.SessionManager
	}

	if config.FallbackPath == "" {
		config.FallbackPath = "/"
	}

	if config.AllowedExtensions == nil {
		config.AllowedExtensions = []string{"", ".html"}
	}

	redirectBack := &RedirectBack{config: config}
	redirectBack.compile()

	middlewares.Use(middlewares.Middleware{
		Name:        "redirect_back",
		InsertAfter: []string{"session"},
		Handler: func(handler http.Handler) http.Handler {
			return redirectBack.Middleware(handler)
		},
	})

	return redirectBack
}

// RedirectBack redirect back struct
type RedirectBack struct {
	config               *Config
	ignoredPathsMap      map[string]bool
	allowedExtensionsMap map[string]bool

	Ignore     func(req *http.Request) bool
	IgnorePath func(pth string) bool
}

func (redirectBack *RedirectBack) compile() {
	redirectBack.ignoredPathsMap = map[string]bool{}

	for _, pth := range redirectBack.config.IgnoredPaths {
		redirectBack.ignoredPathsMap[pth] = true
	}

	redirectBack.allowedExtensionsMap = map[string]bool{}
	for _, ext := range redirectBack.config.AllowedExtensions {
		redirectBack.allowedExtensionsMap[ext] = true
	}

	redirectBack.IgnorePath = func(pth string) bool {
		if !redirectBack.allowedExtensionsMap[filepath.Ext(pth)] {
			return true
		}

		if redirectBack.ignoredPathsMap[pth] {
			return true
		}

		for _, prefix := range redirectBack.config.IgnoredPrefixes {
			if strings.HasPrefix(pth, prefix) {
				return true
			}
		}

		return false
	}

	redirectBack.Ignore = func(req *http.Request) bool {
		if req.Method != "GET" {
			return true
		}

		if redirectBack.config.IgnoreFunc != nil {
			return redirectBack.config.IgnoreFunc(req)
		}

		return redirectBack.IgnorePath(req.URL.Path)
	}
}

// RedirectBack redirect back to last visited page
func (redirectBack *RedirectBack) RedirectBack(w http.ResponseWriter, req *http.Request) {

	// returnTo := req.Context().Value(returnToKey)

	// if returnTo != nil {
	// 	fmt.Println("returnTo1", returnTo)

	// 	fmt.Println("returnTo1", fmt.Sprint(returnTo))
	// 	http.Redirect(w, req, fmt.Sprint(returnTo), http.StatusSeeOther)
	// 	return
	// } else {
	// 	fmt.Println("returnTo2", returnTo)
	// }

	// referrer := req.Referer()

	// if referrer != "" {

	// 	fmt.Println("referrer1", referrer)
	// 	if u, _ := url.Parse(referrer); !redirectBack.IgnorePath(u.Path) {

	// 		fmt.Println("referrer2", referrer)
	// 		http.Redirect(w, req, referrer, http.StatusSeeOther)
	// 		return
	// 	}
	// }

	// fmt.Println("referrer3", referrer)

	// fmt.Println(redirectBack.config.FallbackPath)

	http.Redirect(w, req, redirectBack.config.FallbackPath, http.StatusSeeOther)
}

// Middleware returns a RedirectBack middleware instance that record return_to path
func (redirectBack *RedirectBack) Middleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		returnTo := redirectBack.config.SessionManager.Get(req, "return_to")
		req = req.WithContext(context.WithValue(req.Context(), returnToKey, returnTo))

		if !redirectBack.Ignore(req) && returnTo != req.URL.String() {
			redirectBack.config.SessionManager.Add(w, req, "return_to", req.URL.String())
		}

		handler.ServeHTTP(w, req)
	})
}
