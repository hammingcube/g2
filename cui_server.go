package main

import (
	"bytes"
	"fmt"
	docker_client "github.com/docker/engine-api/client"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	mw "github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/maddyonline/g2/cui"
	"github.com/maddyonline/g2/frontend"
	"github.com/maddyonline/problems"
	"github.com/maddyonline/umpire"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"
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
var problemsList []*problems.Problem

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
		return c.XML(http.StatusOK, cli.GetTask(tasks, getTaskRequest(c)))
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
		resp := cli.GetClock(cuiSessions, clkReq)
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
				resp := cli.GetVerifyStatus(task, solnReq, action.Mode)
				log.Info(action.Path, "\t", "resp: ", resp)
				return c.XML(http.StatusOK, resp)
			}
		}(action)
		chk.Post(action.Path, handler)
	}

	chk.Post("/status", func(c echo.Context) error {
		ticket, verifyKey := c.FormValue("ticket"), c.FormValue("id")
		var resp *cui.VerifyStatus
		cui.Results.Lock()
		resp, ok := cui.Results.Store[fmt.Sprintf("%s/%s", ticket, verifyKey)]
		cui.Results.Unlock()
		if !ok {
			resp = cui.LaterReply(verifyKey)
		}
		return c.XML(http.StatusOK, resp)
	})
}

func refreshProblemsList(problemsDir string, tmpl *template.Template, cli *cui.Client) {
	probsList, err := problems.GetList(problemsDir, ioutil.Discard)
	if err != nil {
		log.Fatal(err)
		return
	}
	index, err := frontend.Index(tmpl, probsList)
	if err != nil {
		log.Fatal(err)
		return
	}
	cli.Mutex.Lock()
	oldCount := len(cli.ProbsList)
	cli.ProbsList = probsList
	cli.Index = index
	cli.LastUpdated = time.Now()
	log.Infof("Updated problems list: %d new problems", len(cli.ProbsList)-oldCount)
	cli.Mutex.Unlock()
}

const PORT = "3000"

var cli *cui.Client

func main() {
	problemsDir, err := filepath.Abs("../../maddyonline/problems")
	if err != nil {
		log.Fatal(err)
		return
	}
	dcli, err := docker_client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	umpireAgent := &umpire.Agent{dcli, problemsDir}

	probsList, err := problems.GetList(problemsDir, ioutil.Discard)
	if err != nil {
		log.Fatal(err)
		return
	}

	tmpl := template.Must(template.ParseFiles(
		"frontend/templates/problems_list.tpl",
		"frontend/templates/main.tpl"))
	index, err := frontend.Index(tmpl, probsList)
	if err != nil {
		log.Fatal(err)
		return
	}

	cli = &cui.Client{
		Agent:       umpireAgent,
		ProbsList:   probsList,
		Index:       index,
		LastUpdated: time.Now(),
		Mutex:       &sync.Mutex{},
	}

	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	defer func() { quit <- struct{}{} }()
	go func() {
		for {
			select {
			case <-ticker.C:
				refreshProblemsList(problemsDir, tmpl, cli)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

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
	e.Get("/", func(c echo.Context) error {
		return c.ServeContent(bytes.NewReader(cli.Index), "index.html", cli.LastUpdated)
	})
	//filepath.Join(rootDir, "frontend/index.html"))
	e.Static("/static/", filepath.Join(rootDir, "frontend/static"))

	// CUI static resources
	e.Static("/static/cui", staticDir)
	t := loadTemplates(templatesDir)
	e.SetRenderer(t)

	// CUI entry point
	e.Get("/cui/new", func(c echo.Context) error {
		problem_id := c.QueryParam("problem_id")
		log.Info(fmt.Sprintf("Got problem_id: %s", problem_id))
		if problem_id == "" {
			return ErrNotFound{}
		}
		ticket := cli.NewTicket(tasks, problem_id)
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
		return c.Render(http.StatusOK, "cui.html", map[string]interface{}{"Title": "Goonj2", "Ticket": session.Ticket})
	})

	// Remaining CUI handlers
	addCuiHandlers(e)

	// Start server
	e.Run(standard.New(fmt.Sprintf(":%s", port)))
}
