package trigger

import (
	"github.com/go-martini/martini"
	"fmt"
	"github.com/emc-cmd/test-framework/scheduler"
)

func RunTriggerServer(sched *scheduler.ExampleScheduler) {
	m := martini.Classic()
	m.Get("/", func() string {
		instructions := fmt.Sprintf("GET / for help\nGET /create/:container_id\nGET /checkpoint/:container_id\nGET /restore/:container_id")
		return instructions
	})
	m.Get("/create/:container_name", func(params martini.Params) string {
		sched.RunContainerTask(params["container_name"])
		return fmt.Sprintf("RunContainerTask queued...\nTask Queue: %v", sched.TaskQueue)
	})
	m.Get("/checkpoint/:container_name", func(params martini.Params) string {
		sched.CheckpointContainerTask(params["container_name"])
		return fmt.Sprintf("CheckpointContainerTask queued...\nTask Queue: %v", sched.TaskQueue)
	})
	m.Get("/restore/:container_name/:target_host", func(params martini.Params) string {
		sched.RestoreContainerTask(params["container_name"], params["target_host"])
		return fmt.Sprintf("RestoreContainerTask queued...\nTask Queue: %v", sched.TaskQueue)
	})


	m.Get("/logs/:container_name", func(params martini.Params) string {
		sched.GetLogsTask(params["container_name"])
		return fmt.Sprintf("GetLogsTask queued...\nTask Queue: %v", sched.TaskQueue)
	})

	m.Run()
}