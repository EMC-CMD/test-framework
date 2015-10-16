package trigger

import (
	"github.com/go-martini/martini"
	"fmt"
	"github.com/emc-cmd/test-framework/scheduler"
)

func RunTriggerServer(sched *scheduler.ExampleScheduler) {
	m := martini.Classic()
	m.Get("/", func() string {
		return "GET /trigger to create and checkpoint a container!\n"
	})
	m.Get("/trigger", func() string {
		sched.AllowedTasks++
		return fmt.Sprintf("now allowing up tp %v tasks", sched.AllowedTasks)
	})
	m.Run()
}