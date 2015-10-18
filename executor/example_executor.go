/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"

	"github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"net/http"
	"bytes"
	"log"
	"io/ioutil"
	"time"
	"math/rand"
	"github.com/emc-cmd/test-framework/containers"
	"github.com/emc-cmd/test-framework/shared"
)

type migrationExecutor struct {
	tasksLaunched int
}

func newExampleExecutor() *migrationExecutor {
	return &migrationExecutor{
		tasksLaunched: 0,
	}
}

func (mExecutor *migrationExecutor) Registered(driver executor.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (mExecutor *migrationExecutor) Reregistered(driver executor.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (mExecutor *migrationExecutor) Disconnected(executor.ExecutorDriver) {
	fmt.Println("Executor disconnected.")
}

func (mExecutor *migrationExecutor) TestRunAndKillContainer(containerName string, url string) {
	container := docker.Docker{
		Name: containerName,
		Image: "busybox:latest",
		Command: `/bin/sh -c 'i=0; while true; do echo "%s: $i"; i=$(expr $i + 1); sleep 1; done'`,
	}

	//run counter in docker container
	out := container.Run()
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes := writeOutputToServer("Initialized docker container: "+out, url)
	fmt.Println("server responded with: "+ string(respBytes))


	//sleep random number of seconds between 5 - 20
	r := rand.New(rand.NewSource(99))
	seconds := r.Int() % 14 + 5
	for i := 0; i < seconds ; i++ {
		fmt.Printf(fmt.Sprintf("Sleeping... %v left", seconds-i-1))
		time.Sleep(1000 * time.Millisecond)
	}

	//read logs from container
	out = container.Logs()
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes = writeOutputToServer(fmt.Sprintf("Slept for "+string(seconds)+"and retrieved logs: %s", out), url)

	//kill & rm container
	out = container.Stop()
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes = writeOutputToServer("Stopped "+containerName+": "+out, url)
	out = container.RM()
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes = writeOutputToServer("Removed "+containerName+": "+out, url)

}


func (mExecutor *migrationExecutor) StartContainer(containerName string, url string) {
	container := docker.Docker{
		Name: containerName,
		Image: "busybox:latest",
		Command: `/bin/sh -c 'i=0; while true; do echo "%s: $i"; i=$(expr $i + 1); sleep 1; done'`,
	}

	//run counter in docker container
	out := container.Run()
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes := writeOutputToServer("Initialized docker container: "+out, url)
	fmt.Println("server responded with: "+ string(respBytes))
}

func (mExecutor *migrationExecutor) CheckpointContainer(containerName string, url string) {
	container := docker.Docker{
		Name: containerName,
		Image: "busybox:latest",
		Command: `/bin/sh -c 'i=0; while true; do echo "%s: $i"; i=$(expr $i + 1); sleep 1; done'`,
	}

	out := container.Export(url)
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes := writeOutputToServer("Checkpointed docker container: "+out, url)
	fmt.Println("server responded with: "+ string(respBytes))
}

func (mExecutor *migrationExecutor) RestoreContainer(containerName string, url string) {
	container := docker.Import(url, containerName)
	respBytes := writeOutputToServer(fmt.Sprintf("Restored docker container: %v", container), url)
	fmt.Println("server responded with: "+ string(respBytes))
}

func (mExecutor *migrationExecutor) LaunchTask(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	fmt.Printf("Launching task %v with data [%#x]\n", taskInfo.GetName(), taskInfo.Data)

	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}

	mExecutor.tasksLaunched++

	/***
	run task
	 ***/

	taskType, err := shared.GetValueFromLabels(taskInfo.Labels, shared.Tags.TASK_TYPE)
	if err != nil {
		fmt.Println("Got error", err)
	}
	url, err := shared.GetValueFromLabels(taskInfo.Labels, shared.Tags.FILESERVER_IP)
	if err != nil {
		fmt.Println("Got error", err)
	}
	containerName, err := shared.GetValueFromLabels(taskInfo.Labels, shared.Tags.CONTAINER_NAME)
	if err != nil {
		fmt.Println("Got error", err)
	}

	switch taskType {
	case shared.TaskTypes.RUN_CONTAINER:
		mExecutor.StartContainer(containerName, url)
		break
		break
	case shared.TaskTypes.CHECKPOINT_CONTAINER:
		mExecutor.CheckpointContainer(containerName, url)
		break
		break
	case shared.TaskTypes.RESTORE_CONTAINER:
		mExecutor.RestoreContainer(containerName, url)
		break
		break
	case shared.TaskTypes.TEST_TASK:
		mExecutor.TestRunAndKillContainer(containerName, url)
		break
	}

	/***
	 finish task
	 ***/
	fmt.Println("Finishing task", taskInfo.GetName())
	finStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		Labels: taskInfo.Labels,
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err = driver.SendStatusUpdate(finStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}
	fmt.Println("Task finished", taskInfo.GetName())
}

func writeOutputToServer(output string, url string) (responseBytes []byte) {
	fmt.Println("Here was the output of the command: "+ output)
	req, _ := http.NewRequest("POST", url+"/in", bytes.NewReader([]byte(fmt.Sprintf(`{"in":"%s"}`, output))))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Got error 2", err)
	}
	responseBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Got error 3", err)
	}
	return
}


func (mExecutor *migrationExecutor) KillTask(executor.ExecutorDriver, *mesos.TaskID) {
	fmt.Println("Kill task")
}

func (mExecutor *migrationExecutor) FrameworkMessage(driver executor.ExecutorDriver, msg string) {
	fmt.Println("Got framework message: ", msg)
}

func (mExecutor *migrationExecutor) Shutdown(executor.ExecutorDriver) {
	fmt.Println("Shutting down the executor")
}

func (mExecutor *migrationExecutor) Error(driver executor.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}

func init() {
	flag.Parse()
}

func main() {
	fmt.Println("Starting Example Executor (Go)")

	dconfig := executor.DriverConfig{
		Executor: newExampleExecutor(),
	}
	driver, err := executor.NewMesosExecutorDriver(dconfig)

	if err != nil {
		fmt.Println("Unable to create a ExecutorDriver ", err.Error())
	}

	_, err = driver.Start()
	if err != nil {
		fmt.Println("Got error:", err)
		return
	}
	fmt.Println("Executor process has started and running.")
	driver.Join()
}
