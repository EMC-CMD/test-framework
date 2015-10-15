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
	"os/exec"
)

type migrationExecutor struct {
	tasksLaunched int
}

func newExampleExecutor() *migrationExecutor {
	return &migrationExecutor{tasksLaunched: 0}
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

//	out, err := exec.Command("echo", "foo").Output()
//	cmdStr := "sudo docker run -v ~/exp/a.out:/a.out ubuntu:14.04 /a.out -m 10m"
	cmdStr := "docker run -t busybox /bin/sh -c 'echo FOOBAR'"
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Got error 1", err, out)
	}

	fmt.Println("Here was the output of the docker run: ", string([]byte(fmt.Sprintf(`{"in":"%s"}`, out))))
	req, _ := http.NewRequest("POST", "http://10.251.232.40:3000/in", bytes.NewReader([]byte(fmt.Sprintf(`{"in":"%s"}`, out))))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Got error 2", err)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Got error 3", err)
	}
	fmt.Println("server responded with: "+ string(bytes))

	//
	// this is where one would perform the requested task
	//

	// finish task
	fmt.Println("Finishing task", taskInfo.GetName())
	finStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err = driver.SendStatusUpdate(finStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}
	fmt.Println("Task finished", taskInfo.GetName())
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
