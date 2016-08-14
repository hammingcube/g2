package main

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	mw "github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/maddyonline/g2/cui"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

type (
	// Template provides HTML template rendering
	Template struct {
		templates *template.Template
	}
)

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func loadTemplates(templatesDir string) *Template {
	t := &Template{
		// Cached templates
		templates: template.Must(template.ParseFiles(filepath.Join(templatesDir, "cui.html"))),
	}
	return t
}

var tasks = map[cui.TaskKey]*cui.Task{}
var cuiSessions = map[string]*cui.Session{}

func getSolutionRequest(c echo.Context) *cui.SolutionRequest {
	return &cui.SolutionRequest{
		Ticket:    c.FormValue("ticket"),
		Task:      c.FormValue("task"),
		ProgLang:  c.FormValue("prg_lang"),
		Solution:  c.FormValue("solution"),
		TestData0: c.FormValue("test_data0"),
	}
}

func getTaskRequest(c echo.Context) *cui.TaskRequest {
	return &cui.TaskRequest{
		Task:                 c.FormValue("task"),
		Ticket:               c.FormValue("ticket"),
		ProgLang:             c.FormValue("prg_lang"),
		HumanLang:            c.FormValue("human_lang"),
		PreferServerProgLang: c.FormValue("prefer_server_prg_lang") == "false",
	}
}

type ErrNotFound struct{}

func (e ErrNotFound) Error() string {
	return "Not Found"
}

func updateTask(solnReq *cui.SolutionRequest) (error, *cui.Task) {
	key := cui.TaskKey{solnReq.Ticket, solnReq.Task}
	task, ok := tasks[key]
	if !ok {
		return ErrNotFound{}, nil
	}
	log.Info(fmt.Sprintf("Updating task (%s): ProgLang from %s to %s", key, task.ProgLang, solnReq.ProgLang))
	log.Info(fmt.Sprintf("Updating task (%s): CurrentSolution from %q to %q", key, task.CurrentSolution, solnReq.Solution))
	task.ProgLang = solnReq.ProgLang
	task.CurrentSolution = solnReq.Solution
	return nil, task
}

func addCuiHandlers(e *echo.Echo) {
	c := e.Group("/c")
	c.Post("/_start", func(c echo.Context) error {
		session, ok := cuiSessions[c.FormValue("ticket")]
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "Attempt to start an invalid session")
		}
		session.StartTime = time.Now()
		return c.String(http.StatusOK, "Started")
	})
	c.Post("/_get_task", func(c echo.Context) error {
		return c.XML(http.StatusOK, cui.GetTask(tasks, getTaskRequest(c)))
	})
	c.Get("/close/:ticket_id", func(c echo.Context) error {
		log.Info("Params: ->%s<-, ->%s<-", c.P(0), c.P(1))
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	})

	chk := e.Group("/chk")
	chk.Post("/clock", func(c echo.Context) error {
		clkReq := &cui.ClockRequest{}
		if err := c.Bind(clkReq); err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Clock Request: %v", clkReq))
		oldlimit := time.Duration(clkReq.OldTimeLimit) * time.Second
		resp := cui.GetClock(cuiSessions, clkReq)
		newlimit := time.Duration(resp.NewTimeLimit) * time.Second
		log.Info(fmt.Sprintf("Clock Request: OldLimit=%s", oldlimit))
		log.Info(fmt.Sprintf("Clock Response: NewLimit=%s", newlimit))
		return c.XML(http.StatusOK, resp)
	})

	chk.Post("/save", func(c echo.Context) error {
		solnReq := getSolutionRequest(c)
		log.Info(fmt.Sprintf("/verify solnReq: %#v", solnReq))
		err, _ := updateTask(solnReq)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, "Finished saving")
	})
	type Action struct {
		Path string
		Mode cui.Mode
	}
	for _, action := range []Action{
		{"/verify", cui.VERIFY},
		{"/judge", cui.JUDGE},
		{"/final", cui.FINAL},
	} {
		handler := func(action Action) echo.HandlerFunc {
			return func(c echo.Context) error {
				solnReq := getSolutionRequest(c)
				log.Info(fmt.Sprintf("%s\tsolnReq: %q", action.Path, solnReq))
				err, task := updateTask(solnReq)
				if err != nil {
					return err
				}
				resp := cui.GetVerifyStatus(task, solnReq, action.Mode)
				log.Info(action.Path, "\t", "resp: ", resp)
				return c.XML(http.StatusOK, resp)
			}
		}(action)
		chk.Post(action.Path, handler)
	}

	chk.Post("/status", func(c echo.Context) error {
		c.FormValue("task")
		//log.Info("/status: %#v", c.Request().Form)
		return c.XML(http.StatusOK, cui.GetVerifyStatus(nil, nil, cui.VERIFY))
	})
}

const PORT = "3000"

func main() {
	rootDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	port := PORT
	staticDir := filepath.Join(rootDir, "static_cui/cui/static/cui")
	templatesDir := filepath.Join(rootDir, "static_cui/cui/templates")

	log.Info(fmt.Sprintf("Using Port=%s", port))
	log.Info(fmt.Sprintf("Using Static Directory=%s", staticDir))
	log.Info(fmt.Sprintf("Using Templates Directory=%s", templatesDir))

	// Echo instance
	e := echo.New()
	e.Pre(mw.RemoveTrailingSlash())

	// Middleware
	e.Use(mw.Logger())
	//e.Use(mw.Recover())
	e.Get("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	// Frontend
	e.File("/", filepath.Join(rootDir, "client-app/index.html"))
	e.Static("/static/", filepath.Join(rootDir, "client-app/static"))

	// CUI static resources
	e.Static("/static/cui", staticDir)
	t := loadTemplates(templatesDir)
	e.SetRenderer(t)

	// CUI entry point
	e.Get("/cui/new/:problem_id", func(c echo.Context) error {
		problem_id := c.Param("problem_id")
		log.Info(fmt.Sprintf("Got problem_id: %s", problem_id))
		ticket := cui.NewTicket(tasks, problem_id)
		cuiSessions[ticket.Id] = &cui.Session{TimeLimit: 3600, Created: time.Now(), Ticket: ticket}
		return c.JSON(http.StatusOK, map[string]string{"ticket_id": ticket.Id, "problem_id": problem_id})
	})
	e.Get("/cui/:ticket_id", func(c echo.Context) error {
		ticket_id := c.Param("ticket_id")
		log.Info("Ticket: %s", ticket_id)
		session, ok := cuiSessions[ticket_id]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound, "No valid session found")
		}
		if !session.Started {
			if time.Now().Sub(session.Created) > time.Duration(10*time.Second) {
				return echo.NewHTTPError(http.StatusNotFound, "Session Expired")
			}
			session.Started = true
		}
		log.Info("Session Started? %v", session.Started)
		return c.Render(http.StatusOK, "cui.html", map[string]interface{}{"Title": "Goonj", "Ticket": session.Ticket})
	})

	// Remaining CUI handlers
	addCuiHandlers(e)

	// Start server
	e.Run(standard.New(fmt.Sprintf(":%s", port)))
}
