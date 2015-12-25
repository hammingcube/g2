package cui

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/maddyonline/goonj/runner"
	"html/template"
	"os"
	"time"
)

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

const tmpl = `var local_ui_options = {{.}}`

func Render(opts *Options) {
	const tpl = `
<!DOCTYPE html>
<html>
    <head>
        <meta charset="UTF-8">
    </head>
    <body>
    <script>var local_ui_options = {{.}}</script>
    </body>
</html>`
	check := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	t, err := template.New("webpage").Parse(tpl)
	check(err)
	err = t.Execute(os.Stdout, opts)
	check(err)
}

type TaskKey struct {
	TicketId string
	TaskId   string
}

type ClientGetTaskMsg struct {
	Task                 string
	Ticket               string
	ProgLang             string
	HumanLang            string
	PreferServerProgLang bool
}

type Session struct {
	StartTime time.Time
	TimeLimit int
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
	Ticket   string `schema:"ticket"`
	Task     string `schema:"task"`
	ProgLang string `scheam:"prg_lang"`
	Solution string `schema:"solution"`
}

type Status struct {
	OK      int    `xml:"ok"`
	Message string `xml:"message"`
}
type MainStatus struct {
	Compile Status `xml:"compile"`
	Example Status `xml:"example"`
}
type VerifyStatus struct {
	XMLName xml.Name   `xml:"response"`
	Result  string     `xml:"result"`
	Extra   MainStatus `xml:"extra"`
	//NextTask string     `xml:"next_task"`
}

func GetVerifyStatus(src string) *VerifyStatus {
	log.Info("In GetVerifyStatus, got src=%s", src)
	resp := &VerifyStatus{
		Result: "OK",
		Extra: MainStatus{
			Compile: Status{1, "The solution compiled flawlessly."},
			Example: Status{1, "OK"},
		},
	}
	if src == "" {
		return resp
	}
	out, err := runner.RunIt(src)
	log.Info("In GetVerifyStatus, got out=%q, err=%v", string(out), err)
	resp.Extra.Example.Message = string(out)
	if err != nil {
		resp.Extra.Example.Message = resp.Extra.Example.Message + "\n" + fmt.Sprintf("%s", err)
		resp.Extra.Example.OK = 0
	} else {
		resp.Extra.Example.OK = 1
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

func GetTask(tasks map[TaskKey]*Task, val *ClientGetTaskMsg) *Task {
	key := TaskKey{val.Ticket, val.Task}
	prg_lang_list, _ := json.Marshal([]string{"c", "cpp"})
	human_lang_list, _ := json.Marshal([]string{"en", "cn"})
	task := tasks[key]
	if task == nil {
		log.Info("Serving task based on nil request")
		task = &Task{
			Id:               val.Task,
			Status:           "open",
			Description:      "Description: task1,en,c",
			Type:             "algo",
			SolutionTemplate: "",
			CurrentSolution:  "",
			ExampleInput:     "",
			ProgLangList:     string(prg_lang_list),
			HumanLangList:    string(human_lang_list),
			ProgLang:         val.ProgLang,
			HumanLang:        val.HumanLang,
		}
		tasks[key] = task
	}
	log.Info("Updating task %s prog-lang form %s to %s", task.Id, task.ProgLang, val.ProgLang)
	log.Info("Updating task %s prog-lang form %s to %s", task.Id, task.HumanLang, val.HumanLang)
	task.ProgLang = val.ProgLang
	task.HumanLang = val.HumanLang
	return task
}

const expected = `{
        ticket_id: "TICKET_ID",

        time_elapsed_sec: 15,
        time_remaining_sec: 1800,

        current_human_lang: "en",
        current_prg_lang: "c",
        current_task_name: "task1",

        task_names: ["task1", "task2", "task3"],

        human_langs: {
            "en": {"name_in_itself": "English"},
            "cn": {"name_in_itself": "\u4e2d\u6587"},
        },
        prg_langs: {
            "c": {"version": "C", "name": "C"},
            "sql": {"version": "SQL", "name": "SQL"},
            "cpp": {"version": "C++", "name": "C++"},
        },

        show_survey: true,
        show_help: false,
        show_welcome: true,
        sequential: false,
        save_often: true,

        urls: {
            "status": "/chk/status/",
            "get_task": "/c/_get_task/",
            "submit_survey": "/surveys/_ajax_submit_candidate_survey/TICKET_ID/",
            "clock": "/chk/clock/",
            "close": "/c/close/TICKET_ID",
            "verify": "/chk/verify/",
            "save": "/chk/save/",
            "timeout_action": "/chk/timeout_action/",
            "final": "/chk/final/",
            "start_ticket": "/c/_start/"
        },
        }`
