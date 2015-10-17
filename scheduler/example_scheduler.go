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

package scheduler

import (
	"github.com/gogo/protobuf/proto"
	"strconv"

	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"github.com/emc-cmd/test-framework/shared"
)

type ExampleScheduler struct {
	executor      *mesos.ExecutorInfo
	tasksLaunched int
	tasksFinished int
	totalTasks    int
	cpuPerTask    float64
	memPerTask    float64
	TaskQueue	[]*mesos.TaskInfo
}

func NewExampleScheduler(exec *mesos.ExecutorInfo, taskCount int, cpuPerTask float64, memPerTask float64) *ExampleScheduler {
	return &ExampleScheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    taskCount,
		cpuPerTask:    cpuPerTask,
		memPerTask:    memPerTask,
	}
}

func (sched *ExampleScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Scheduler Registered with Master ", masterInfo)
}

func (sched *ExampleScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Scheduler Re-Registered with Master ", masterInfo)
}

func (sched *ExampleScheduler) Disconnected(sched.SchedulerDriver) {
	log.Infoln("Scheduler Disconnected")
}

func (sched *ExampleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	logOffers(offers)
	log.Infof("received some offers, but do I care?")

	for _, offer := range offers {
		remainingCpus := getOfferCpu(offer)
		remainingMems := getOfferMem(offer)

		var tasks []*mesos.TaskInfo
		for sched.cpuPerTask <= remainingCpus &&
		sched.memPerTask <= remainingMems &&
		len(sched.TaskQueue) > 0 {

			log.Infof("Launched tasks: "+string(sched.tasksLaunched))
			log.Infof("Tasks remaining ot be launched: "+string(len(sched.TaskQueue)))

			sched.tasksLaunched++

			task := sched.popTask()
			task.SlaveId = offer.SlaveId
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			remainingCpus -= sched.cpuPerTask
			remainingMems -= sched.memPerTask
		}
		log.Infoln("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *ExampleScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())
}

func (sched *ExampleScheduler) OfferRescinded(s sched.SchedulerDriver, id *mesos.OfferID) {
	log.Infof("Offer '%v' rescinded.\n", *id)
}

func (sched *ExampleScheduler) FrameworkMessage(s sched.SchedulerDriver, exId *mesos.ExecutorID, slvId *mesos.SlaveID, msg string) {
	log.Infof("Received framework message from executor '%v' on slave '%v': %s.\n", *exId, *slvId, msg)
}

func (sched *ExampleScheduler) SlaveLost(s sched.SchedulerDriver, id *mesos.SlaveID) {
	log.Infof("Slave '%v' lost.\n", *id)
}

func (sched *ExampleScheduler) ExecutorLost(s sched.SchedulerDriver, exId *mesos.ExecutorID, slvId *mesos.SlaveID, i int) {
	log.Infof("Executor '%v' lost on slave '%v' with exit code: %v.\n", *exId, *slvId, i)
}

func (sched *ExampleScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}


func (sched *ExampleScheduler) RunContainerTask(containerID string) {
	log.Infoln("Generating RUN_CONTAINER task...")
	tags := map[string]string{
		shared.TASK_TYPE : shared.RUN_CONTAINER,
		shared.CONTAINER_NAME: containerID,
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) CheckpointContainerTask(containerID string) {
	log.Infoln("Generating CHECKPOINT_CONTAINER task...")
	tags := map[string]string{
		shared.TASK_TYPE : shared.CHECKPOINT_CONTAINER,
		shared.CONTAINER_NAME: containerID,
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) RestoreContainerTask(containerID string) {
	log.Infoln("Generating RESTORE_CONTAINER task...")
	tags := map[string]string{
		shared.TASK_TYPE : shared.RESTORE_CONTAINER,
		shared.CONTAINER_NAME: containerID,
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) pushTask(task *mesos.TaskInfo) {
	sched.TaskQueue = append(sched.TaskQueue, task)
}

func (sched *ExampleScheduler) popTask() *mesos.TaskInfo{
	task := sched.TaskQueue[len(sched.TaskQueue)-1]
	sched.TaskQueue = sched.TaskQueue[:len(sched.TaskQueue)-1]
	return task
}

func (sched *ExampleScheduler) genTask(tags map[string]string) *mesos.TaskInfo {
	taskId := &mesos.TaskID{
		Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
	}
	labels := &mesos.Labels{
		Labels: []*mesos.Label{
		},
	}
	for key, value := range tags {
		label := &mesos.Label{
			Key: &key,
			Value: &value,
		}
		labels.Labels = append(labels.Labels, label)
	}
	task := &mesos.TaskInfo{
		Name:     proto.String("go-task-" + taskId.GetValue()),
		TaskId:   taskId,
		Executor: sched.executor,
		Resources: []*mesos.Resource{
			util.NewScalarResource("cpus", sched.cpuPerTask),
			util.NewScalarResource("mem", sched.memPerTask),
		},
		Labels: labels,
	}
	return task
}