package service

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/cfg"
)

type Server struct {
	srvr    http.Server
	name    string
	port    string      // port server is running on; must have leading :, as in ":3000"
	version string      // version running on server
	info    interface{} // json metadata about server
	router  chi.Router
	unsafe  bool		// running in http mode   // TODO set this from HTTP_ONLY envv
}

func NewServer(name, port, version string, info interface{}, routes *chi.Mux, unsafe bool) *Server {
	s := Server{}
	s.name = name
	s.port = port
	s.version = version
	s.info = info
	s.router = s.newBaseRouter()
	s.router.Mount("/", routes)
	s.unsafe = unsafe
	s.srvr = http.Server{
		Handler:      s.router,
		Addr:         port,
		ReadTimeout:  time.Duration(cfg.GetEnvInt("SSAS_READ_TIMEOUT", 10)) * time.Second,
		WriteTimeout: time.Duration(cfg.GetEnvInt("SSAS_WRITE_TIMEOUT", 20)) * time.Second,
		IdleTimeout:  time.Duration(cfg.GetEnvInt("SSAS_IDLE_TIMEOUT", 120)) * time.Second,
	}

	return &s
}

// only used for creation of Server instance
func (s *Server) newBaseRouter() *chi.Mux {
	r := chi.NewRouter()
	// TODO middlewares here, eg monitoring, logging
	r.Use(
		NewAPILogger(),
		render.SetContentType(render.ContentTypeJSON),
		ConnectionClose,
	)
	r.Get("/_version", s.getVersion)
	r.Get("/_health", s.getHealthCheck)
	r.Get("/_info", s.getInfo)
	return r
}

// https://itnext.io/structuring-a-production-grade-rest-api-in-golang-c0229b3feedc
func (s *Server) LogRoutes() {
	routes := fmt.Sprintf("Routes for %s at port %s: ", s.name, s.port)
	walker := func(method, route string, handler http.Handler, middlewares ...func (http.Handler) http.Handler) error {
		routes = fmt.Sprintf("%s %s %s, ", routes, method, route)
		return nil
	}
	if err := chi.Walk(s.router, walker); err != nil {
		ssas.Logger.Fatalf("bad route: %s", err.Error())
	}
	ssas.Logger.Infof(routes)
}

func (s *Server) Serve() {
	tlsCertPath := os.Getenv("BCDA_TLS_CERT")	// borrowing for now; we need to get our own
	tlsKeyPath := os.Getenv("BCDA_TLS_KEY")

	if (s.unsafe) {
		ssas.Logger.Infof("starting %s server running UNSAFE http only mode; do not do this in production environments", s.name)
		go func() { log.Fatal(s.srvr.ListenAndServe()) }()
	} else {
		go func() { log.Fatal(s.srvr.ListenAndServeTLS(tlsCertPath, tlsKeyPath))}()
	}
}

func (s *Server) getInfo(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, s.info)
}

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	respMap := make(map[string]string)
	respMap["version"] = fmt.Sprintf("%v", s.version)
	render.JSON(w, r, s.version)
}

func (s *Server) getHealthCheck(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]string)
	if doHealthCheck() {
		m["database"] = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		m["database"] = "error"
		w.WriteHeader(http.StatusBadGateway)
	}
	render.JSON(w, r, m)
}

// is this the right health check for this service? the db could be up but the service down
func doHealthCheck() bool {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		ssas.Logger.Error("health check: database connection error: ", err.Error())
		return false
	}

	defer func() {
		if err = db.Close(); err != nil {
			ssas.Logger.Infof("failed to close db connection in ssas/service/server.go#doHealthCheck() because %s", err)
		}
	}()

	if err = db.Ping(); err != nil {
		ssas.Logger.Error("health check: database ping error: ", err.Error())
		return false
	}

	return true
}

// NYI provides a convenient handler for endpoints that are not yet implemented
func NYI(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string)
	response["msg"] = "Not Yet Implemented"
	render.JSON(w, r, response)
}

func ConnectionClose(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		next.ServeHTTP(w, r)
	})
}