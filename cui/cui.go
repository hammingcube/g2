package cui

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/maddyonline/problems"
	"github.com/maddyonline/umpire"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"strings"
	"sync"
	"time"
)

func RandId(size int) string {
	rb := make([]byte, size)
	_, err := rand.Read(rb)

	if err != nil {
		fmt.Println(err)
	}

	rs := base64.URLEncoding.EncodeToString(rb)

	return rs
}

type Client struct {
	Agent       *umpire.Agent
	ProbsList   map[string]*problems.Problem
	Index       []byte
	LastUpdated time.Time
	*sync.Mutex
}

type HumanLang struct {
	Name string `json:"name_in_itself"`
}
type ProgLang struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Options struct {
	TicketId         string               `json:"ticket_id"`
	TimeElapsed      int                  `json:"time_elpased_sec"`
	TimeRemaining    int                  `json:"time_remaining_sec"`
	CurrentHumanLang string               `json:"current_human_lang"`
	CurrentProgLang  string               `json:"current_prg_lang"`
	CurrentTaskName  string               `json:"current_task_name"`
	TaskNames        []string             `json:"task_names"`
	HumanLangList    map[string]HumanLang `json:"human_langs"`
	ProgLangList     map[string]ProgLang  `json:"prg_langs"`
	ShowSurvey       bool                 `json:"show_survey"`
	ShowHelp         bool                 `json:"show_help"`
	ShowWelcome      bool                 `json:"show_welcome"`
	Sequential       bool                 `json:"sequential"`
	SaveOften        bool                 `json:"save_often"`
	Urls             map[string]string    `json:"urls"`
}

type Ticket struct {
	Id      string
	Options *Options
}

type Session struct {
	Ticket    *Ticket
	StartTime time.Time
	Created   time.Time
	Started   bool
	TimeLimit int
}

type TaskKey struct {
	TicketId string
	TaskId   string
}

type TaskRequest struct {
	Task                 string
	Ticket               string
	ProgLang             string
	HumanLang            string
	PreferServerProgLang bool
}

type Task struct {
	XMLName          xml.Name          `xml:"response"`
	Id               string            `xml:"id" json:"id"`
	Status           string            `xml:"task_status" json: "task_status"`
	Description      string            `xml:"task_description"`
	Type             string            `xml:"task_type"`
	SolutionTemplate string            `xml:"solution_template"`
	CurrentSolution  string            `xml:"current_solution"`
	ExampleInput     string            `xml:"example_input"`
	ProgLangList     string            `xml:"prg_lang_list"`
	HumanLangList    string            `xml:"human_lang_list"`
	ProgLang         string            `xml:"prg_lang"`
	HumanLang        string            `xml:"human_lang"`
	Src              string            `xml:"-"`
	Filename         string            `xml:"-"`
	Templates        map[string]string `xml:"-"`
}

type ClockRequest struct {
	TicketId     string `schema:"ticket"`
	OldTimeLimit int    `schema:"old_timelimit"`
}
type ClockResponse struct {
	XMLName      xml.Name `xml:"response"`
	Result       string   `xml:"result"`
	NewTimeLimit int      `xml:"new_timelimit"`
}

type SolutionRequest struct {
	Ticket    string `schema:"ticket"`
	Task      string `schema:"task"`
	ProgLang  string `scheam:"prg_lang"`
	Solution  string `schema:"solution"`
	TestData0 string `schema:"test_data0"`
	TestData1 string `schema:"test_data1"`
	TestData2 string `schema:"test_data2"`
	TestData3 string `schema:"test_data3"`
	TestData4 string `schema:"test_data4"`
}

type Status struct {
	OK      int    `xml:"ok"`
	Message string `xml:"message"`
}
type MainStatus struct {
	Compile   Status `xml:"compile"`
	Example   Status `xml:"example"`
	TestData0 Status `xml:"test_data0"`
	TestData1 Status `xml:"test_data1"`
	TestData2 Status `xml:"test_data2"`
	TestData3 Status `xml:"test_data3"`
	TestData4 Status `xml:"test_data4"`
}
type VerifyStatus struct {
	XMLName xml.Name   `xml:"response"`
	Result  string     `xml:"result"`
	Message string     `xml:"message"`
	Id      string     `xml:"id"`
	Delay   int        `xml:"delay"`
	Extra   MainStatus `xml:"extra"`
	//NextTask string     `xml:"next_task"`
}

var CUI_LANG_TO_MD = map[string]string{
	"cpp": "cpp",
	"py2": "python",
	"py3": "python",
}

func (client *Client) NewTicket(tasks map[TaskKey]*Task, taskId string) *Ticket {
	ticketId := RandId(32)
	task := NewTask()
	task.Id = taskId
	prob := client.ProbsList[taskId]

	log.Infof("prob: %+v", prob)

	task.Description = string(getDescFromMarkdown([]byte(prob.FullDesc)))
	task.Templates = prob.Templates
	task.SolutionTemplate = prob.Templates[CUI_LANG_TO_MD[task.ProgLang]]
	task.CurrentSolution = prob.Templates[CUI_LANG_TO_MD[task.ProgLang]]
	tasks[TaskKey{ticketId, taskId}] = task
	log.Infof("%#v", tasks)
	opts := DefaultOptions()
	opts.TicketId = ticketId
	taskIds := []string{task.Id}
	opts.TaskNames = taskIds
	opts.CurrentTaskName = task.Id
	opts.CurrentProgLang = task.ProgLang
	opts.Urls["close"] = strings.Replace(opts.Urls["close"], "TICKET_ID", opts.TicketId, -1)
	opts.Urls["submit_survey"] = strings.Replace(opts.Urls["submit_survey"], "TICKET_ID", opts.TicketId, -1)
	return &Ticket{Id: ticketId, Options: opts}
}

func DefaultHumanLangList() map[string]HumanLang {
	return map[string]HumanLang{
		"en": HumanLang{Name: "English"},
		"cn": HumanLang{Name: "\u4e2d\u6587"},
	}
}

func DefaultProgLangList() map[string]ProgLang {
	return map[string]ProgLang{
		"c":   ProgLang{Version: "C", Name: "C"},
		"cpp": ProgLang{Version: "C++", Name: "C++"},
		"py2": ProgLang{Version: "py2", Name: "Python 2"},
		"py3": ProgLang{Version: "py3", Name: "Python 3"},
		"go":  ProgLang{Version: "go", Name: "Go"},
		"js":  ProgLang{Version: "js", Name: "Javascript"},
	}
}

func DefaultOptions() *Options {
	opts := &Options{
		TicketId:         "",
		TimeElapsed:      5,
		TimeRemaining:    3600,
		CurrentHumanLang: "en",
		CurrentProgLang:  "c",
		CurrentTaskName:  "task1",
		TaskNames:        []string{"task1", "task2", "task3"},
		HumanLangList:    DefaultHumanLangList(),
		ProgLangList:     DefaultProgLangList(),
		ShowSurvey:       false,
		ShowWelcome:      false,
		Sequential:       false,
		SaveOften:        true,
		Urls: map[string]string{
			"status":         "/chk/status/",
			"get_task":       "/c/_get_task/",
			"submit_survey":  "/surveys/_ajax_submit_candidate_survey/TICKET_ID/",
			"clock":          "/chk/clock/",
			"close":          "/c/close/TICKET_ID",
			"verify":         "/chk/verify/",
			"judge":          "/chk/judge/",
			"save":           "/chk/save/",
			"timeout_action": "/chk/timeout_action/",
			"final":          "/chk/final/",
			"start_ticket":   "/c/_start/",
		},
	}
	return opts
}

type Mode int

const (
	VERIFY Mode = iota
	JUDGE
	FINAL
)

func (t Mode) String() string {
	var val string
	switch t {
	case VERIFY:
		val = "VERIFY"
	case JUDGE:
		val = "JUDGE"
	case FINAL:
		val = "FINAL"
	}
	return val
}

func errorReply(err error, v *VerifyStatus) *VerifyStatus {
	v.Extra.Compile.OK = 0
	v.Extra.Compile.Message = fmt.Sprintf("Something went wrong: %v", err)
	v.Extra.Example.OK = 0
	v.Extra.Example.Message = "Something went wrong"
	return v
}

func LaterReply(key string) *VerifyStatus {
	log.Info("laterReply")
	resp := &VerifyStatus{
		Result:  "LATER",
		Message: "We are still evaluating the solution",
		Id:      key,
		Delay:   60,
	}
	return resp
}

type ResultStore struct {
	Store map[string]*VerifyStatus
	*sync.Mutex
}

var Results = &ResultStore{
	Store: map[string]*VerifyStatus{},
	Mutex: &sync.Mutex{},
}

func getPayload(task *Task, solnReq *SolutionRequest) *umpire.Payload {
	return &umpire.Payload{
		Problem:  &umpire.Problem{task.Id},
		Language: "cpp",
		Files: []*umpire.InMemoryFile{
			&umpire.InMemoryFile{
				Name:    "main.cpp",
				Content: task.CurrentSolution,
			},
		},
		Stdin: solnReq.TestData0,
	}
}

func defaultVerifyStatus() *VerifyStatus {
	return &VerifyStatus{
		Result: "OK",
		Extra: MainStatus{
			Compile:   Status{1, "The solution compiled flawlessly."},
			Example:   Status{1, "OK"},
			TestData0: Status{1, "OK"},
			TestData1: Status{1, "OK"},
			TestData2: Status{1, "OK"},
			TestData3: Status{1, "OK"},
			TestData4: Status{1, "OK"},
		},
	}
}

func (client *Client) GetVerifyStatus(task *Task, solnReq *SolutionRequest, mode Mode) *VerifyStatus {
	log.Infof("In VerifyStatus, mode=%s", mode)
	verifyKey := RandId(4)
	log.Infof("Client: %+v", client)
	agent := client.Agent
	payload := getPayload(task, solnReq)
	done := make(chan *VerifyStatus)
	go func() {
		var out *umpire.Response
		switch mode {
		case VERIFY:
			out = umpire.RunDefault(agent, payload)
		case JUDGE, FINAL:
			out = umpire.JudgeDefault(agent, payload)
		}
		resp := defaultVerifyStatus()
		msg, _ := json.Marshal(out)
		resp.Extra.Example.Message = out.Stdout + "\n" + out.Stderr + "\n" + out.Details + "\n" + string(msg)
		if out.Status == umpire.Fail {
			resp.Extra.Example.OK = 0
		}
		Results.Lock()
		Results.Store[fmt.Sprintf("%s/%s", solnReq.Ticket, verifyKey)] = resp
		Results.Unlock()
		done <- resp
	}()
	for {
		select {
		case resp := <-done:
			return resp
		case <-time.After(1 * time.Second):
			go func() { <-done }()
			return LaterReply(verifyKey)
		}
	}
}

func (client *Client) GetClock(sessions map[string]*Session, clkReq *ClockRequest) *ClockResponse {
	session, ok := sessions[clkReq.TicketId]
	if !ok {
		return &ClockResponse{Result: "OK", NewTimeLimit: clkReq.OldTimeLimit}
	}
	elapsed := int(time.Since(session.StartTime) / time.Second)
	remaining := session.TimeLimit - elapsed
	log.Info("elapsed: %s, remaining: %s", time.Duration(elapsed)*time.Second, time.Duration(remaining)*time.Second)
	if remaining < 0 {
		remaining = 0
	}
	log.Info("newTimeLimit: %v, that is, %s", remaining, time.Duration(remaining)*time.Second)
	return &ClockResponse{Result: "OK", NewTimeLimit: remaining}
}

const SOLN_TEMPL_CPP = `# include <iostream>
using namespace std;
int main() {
  string s;
  while(cin >> s) {
    cout << s.size() << endl;
  }
}
`

const DESC_TEMPL = `### Can a string be a palindrome?

Given a string (say appease) be converted into a palindrome by permuting it?
In this case, we see appease --> apesepa is possible.

What is a good algorithm for this?

#### Test Cases

#### Input

	appease
	appeal

#### Output

	1
	0


### Code Example

	func getTrue() bool {
	    return true
	}
`

func getDescFromMarkdown(input []byte) []byte {
	unsafe := blackfriday.MarkdownCommon(input)
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return html
}

func NewTask() *Task {
	task := &Task{
		Id:               "",
		Status:           "open",
		Description:      "Placeholder",
		Type:             "algo",
		SolutionTemplate: "This is just a template",
		CurrentSolution:  "",
		ExampleInput:     "",
		ProgLangList:     ProgrammingLanguageList(),
		HumanLangList:    HumanLanguageList(),
		ProgLang:         "cpp",
		HumanLang:        "en",
	}
	return task
}

func ProgrammingLanguageList() string {
	list := DefaultProgLangList()
	keys := []string{}
	for k := range list {
		keys = append(keys, k)
	}
	log.Printf("Loading following languages: %v", keys)
	prg_lang_list, _ := json.Marshal(keys)
	return string(prg_lang_list)
}

func HumanLanguageList() string {
	human_lang_list, _ := json.Marshal([]string{"en", "cn"})
	return string(human_lang_list)
}

func (client *Client) GetTask(tasks map[TaskKey]*Task, msg *TaskRequest) *Task {
	key := TaskKey{msg.Ticket, msg.Task}
	task, ok := tasks[key]
	log.Info(fmt.Sprintf("Looking for %s in tasks: %v", key, ok))

	if !ok || task == nil {
		log.Info("Serving task based on nil request")
		task = &Task{
			Id:               msg.Task,
			Status:           "open",
			Description:      string(getDescFromMarkdown([]byte(DESC_TEMPL))),
			Type:             "algo",
			SolutionTemplate: "This is just a template",
			CurrentSolution:  SOLN_TEMPL_CPP,
			ExampleInput:     "",
			ProgLangList:     ProgrammingLanguageList(),
			HumanLangList:    HumanLanguageList(),
			ProgLang:         msg.ProgLang,
			HumanLang:        msg.HumanLang,
		}
		tasks[key] = task
	}
	log.Info(fmt.Sprintf("PREFER-SERVER-LANG: %v", msg.PreferServerProgLang))
	if msg.PreferServerProgLang {
		log.Info(fmt.Sprintf("Updating task %s prog-lang form %s to %s", task.Id, task.ProgLang, msg.ProgLang))
		task.ProgLang = msg.ProgLang
	}
	log.Info(fmt.Sprintf("Updating task %s prog-lang form %s to %s", task.Id, task.HumanLang, msg.HumanLang))
	task.HumanLang = msg.HumanLang
	task.SolutionTemplate = task.Templates[CUI_LANG_TO_MD[task.ProgLang]]
	return task
}
