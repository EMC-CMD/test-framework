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
	"time"
	"math/rand"
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

	/***
	run task
	 ***/

	containerName := fmt.Sprintf("migrate-me-%v", mExecutor.tasksLaunched)
	url := "http://192.168.0.15:3000/in"

	//run counter in docker container
	cmdStr := fmt.Sprintf(`docker run -d --name %s busybox:latest /bin/sh -c 'i=0; while true; do echo "%s: $i"; i=$(expr $i + 1); sleep 1; done'`, containerName, containerName)
	out := runCommand(cmdStr)
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
	cmdStr = fmt.Sprintf(`docker logs %s`, containerName)
	out = runCommand(cmdStr)
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes = writeOutputToServer(fmt.Sprintf("Slept for "+string(seconds)+"and retrieved logs: %s", out), url)

	//kill & rm container
	cmdStr = fmt.Sprintf(`docker stop %s`, containerName)
	out = runCommand(cmdStr)
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes = writeOutputToServer("Stopped "+containerName+": "+out, url)
	cmdStr = fmt.Sprintf(`docker rm %s`, containerName)
	out = runCommand(cmdStr)
	out = out[0:len(out)-2] //for some reason necessary?
	respBytes = writeOutputToServer("Removed "+containerName+": "+out, url)

	/***
	 finish task
	 ***/
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

func runCommand(command string) string {
	cmdStr := fmt.Sprintf(`%s`, command)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Got error running command: ", cmdStr)
	}
	return string(out)
}

func writeOutputToServer(output string, url string) (responseBytes []byte) {
	fmt.Println("Here was the output of the command: "+ output)
	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(fmt.Sprintf(`{"in":"%s"}`, output))))
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
