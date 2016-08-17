package cui

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/labstack/gommon/log"
	"github.com/maddyonline/umpire"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

func RandId() string {
	size := 32 // change the length of the generated random string here

	rb := make([]byte, size)
	_, err := rand.Read(rb)

	if err != nil {
		fmt.Println(err)
	}

	rs := base64.URLEncoding.EncodeToString(rb)

	return rs
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
	XMLName          xml.Name `xml:"response"`
	Id               string   `xml:"id" json:"id"`
	Status           string   `xml:"task_status" json: "task_status"`
	Description      string   `xml:"task_description"`
	Type             string   `xml:"task_type"`
	SolutionTemplate string   `xml:"solution_template"`
	CurrentSolution  string   `xml:"current_solution"`
	ExampleInput     string   `xml:"example_input"`
	ProgLangList     string   `xml:"prg_lang_list"`
	HumanLangList    string   `xml:"human_lang_list"`
	ProgLang         string   `xml:"prg_lang"`
	HumanLang        string   `xml:"human_lang"`
	Src              string   `xml:"-"`
	Filename         string   `xml:"-"`
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

func NewTicket(tasks map[TaskKey]*Task, taskId string) *Ticket {
	ticketId := RandId()
	task := NewTask()
	task.Id = taskId
	desc, err := ioutil.ReadFile(fmt.Sprintf("../../%s/README.md", taskId))
	if err != nil {
		log.Printf("%v", err)
		return nil
	}
	task.Description = string(getDescFromMarkdown(desc))
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

func DefaultOptions() *Options {
	opts := &Options{
		TicketId:         "",
		TimeElapsed:      5,
		TimeRemaining:    3600,
		CurrentHumanLang: "en",
		CurrentProgLang:  "c",
		CurrentTaskName:  "task1",
		TaskNames:        []string{"task1", "task2", "task3"},
		HumanLangList: map[string]HumanLang{
			"en": HumanLang{Name: "English"},
			"cn": HumanLang{Name: "\u4e2d\u6587"},
		},
		ProgLangList: map[string]ProgLang{
			"c":   ProgLang{Version: "C", Name: "C"},
			"cpp": ProgLang{Version: "C++", Name: "C++"},
			"py2": ProgLang{Version: "py2", Name: "Python 2"},
			"py3": ProgLang{Version: "py3", Name: "Python 3"},
			"go":  ProgLang{Version: "go", Name: "Go"},
			"js":  ProgLang{Version: "js", Name: "Javascript"},
		},
		ShowSurvey:  false,
		ShowWelcome: false,
		Sequential:  false,
		SaveOften:   true,
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

func laterReply() *VerifyStatus {
	log.Info("laterReply")
	resp := &VerifyStatus{
		Result:  "LATER",
		Message: "We are still evaluating the solution",
		Id:      "submission_id: 23e3",
		Delay:   60,
	}
	return resp
}

func GetVerifyStatus(task *Task, solnReq *SolutionRequest, mode Mode) *VerifyStatus {
	log.Info("In VerifyStatus, mode=%s", mode)
	//return laterReply()
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("%v", err)
		return nil
	}
	problemsDir, err := filepath.Abs("../../")
	if err != nil {
		log.Fatalf("%v", err)
		return nil
	}
	u := &umpire.Umpire{cli, problemsDir}
	payload := &umpire.Payload{
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
	out := umpire.RunDefault(u, payload)
	msg, _ := json.Marshal(out)
	resp := &VerifyStatus{
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
	// errorResponse(err, resp)
	switch mode {
	case VERIFY:
		resp.Extra.Example.OK = 1
		resp.Extra.Example.Message = string(msg)
	case JUDGE, FINAL:
		resp.Extra.Example.OK = 1
		resp.Extra.Example.Message = "All well in judge/final"
	}
	return resp
}

func GetClock(sessions map[string]*Session, clkReq *ClockRequest) *ClockResponse {
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
	prg_lang_list, _ := json.Marshal([]string{"c", "cpp", "py2", "py3", "go", "js"})
	return string(prg_lang_list)
}

func HumanLanguageList() string {
	human_lang_list, _ := json.Marshal([]string{"en", "cn"})
	return string(human_lang_list)
}

func GetTask(tasks map[TaskKey]*Task, msg *TaskRequest) *Task {
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
	return task
}
