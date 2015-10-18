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
	ContainerSlaveMap map[string]string //map of Container name to hostname
	ExternalServer string

}

func NewExampleScheduler(exec *mesos.ExecutorInfo, taskCount int, cpuPerTask float64, memPerTask float64, ip string) *ExampleScheduler {
	return &ExampleScheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    taskCount,
		cpuPerTask:    cpuPerTask,
		memPerTask:    memPerTask,
		ExternalServer: ip,
		ContainerSlaveMap: make(map[string]string),
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
			taskType, err := shared.GetValueFromLabels(task.Labels, shared.Tags.TASK_TYPE)
			if err != nil{
				log.Infof("ERROR: Malformed task info, discarding task %v", task)
				return
			}
			targetHost, err := shared.GetValueFromLabels(task.Labels, shared.Tags.TARGET_HOST)
			if err != nil && taskType != shared.TaskTypes.RUN_CONTAINER {
				log.Infof("ERROR: Malformed task info, discarding task %v", task)
				return
			}
			containerName, err := shared.GetValueFromLabels(task.Labels, shared.Tags.CONTAINER_NAME)
			if err != nil{
				log.Infof("ERROR: Malformed task info, discarding task %v", task)
				return
			}

			foundAMatch := false
			switch taskType{
			case shared.TaskTypes.RESTORE_CONTAINER:
				if _, ok := sched.ContainerSlaveMap[containerName]; ok {
					log.Infof("ERROR: %s is already running", containerName)
					return
				}
				foundAMatch = true
				break
			case shared.TaskTypes.GET_LOGS:
				if targetHost == offer.GetHostname() {
					foundAMatch = sched.ContainerSlaveMap[containerName] == targetHost
				}
				break
			case shared.TaskTypes.CHECKPOINT_CONTAINER:
				if targetHost == offer.GetHostname() {
					foundAMatch = sched.ContainerSlaveMap[containerName] == targetHost
				}
				break
			case shared.TaskTypes.RESTORE_CONTAINER:
				if _, ok := sched.ContainerSlaveMap[containerName]; ok {
					log.Infof("ERROR: %s is already running", containerName)
					return
				}
				if targetHost == offer.GetHostname() {
					foundAMatch = true
				}
				break
			default:
				foundAMatch = true
				break
			}
			if foundAMatch {
				task.SlaveId = offer.SlaveId
				task.Labels.Labels = append(task.Labels.Labels, shared.CreateLabel(shared.Tags.ACCEPTED_HOST, *offer.Hostname))
				log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

				tasks = append(tasks, task)
				remainingCpus -= sched.cpuPerTask
				remainingMems -= sched.memPerTask
			} else {
				defer sched.pushTask(task)
			}
		}
		log.Infoln("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue(), "\nSlaveID: ", offer.GetSlaveId(),"SlaveHostname: ", offer.GetHostname())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *ExampleScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())
	//if RunContainer finished, add
	if status.State.Enum().String() == "TASK_FINISHED" {
		labels := status.GetLabels()
		taskType, err := shared.GetValueFromLabels(labels, shared.Tags.TASK_TYPE)
		if err != nil{
			log.Infof("ERROR: Malformed task info, discarding task with status: %v", status)
			return
		}
		acceptedHost, err := shared.GetValueFromLabels(labels, shared.Tags.ACCEPTED_HOST)
		if err != nil{
			log.Infof("ERROR: Malformed task info, discarding task with status: %v", status)
			return
		}
		containerName, err := shared.GetValueFromLabels(labels, shared.Tags.CONTAINER_NAME)
		if err != nil{
			log.Infof("ERROR: Malformed task info, discarding task with status: %v", status)
			return
		}
		switch taskType {
		case shared.TaskTypes.RUN_CONTAINER:
			sched.ContainerSlaveMap[containerName] = acceptedHost
			break
		case shared.TaskTypes.CHECKPOINT_CONTAINER:
			delete(sched.ContainerSlaveMap, containerName)
			break
		case shared.TaskTypes.RESTORE_CONTAINER:
			sched.ContainerSlaveMap[containerName] = acceptedHost
			break
		}
	}
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

func (sched *ExampleScheduler) TestTask(containerID string) {
	log.Infoln("Generating RUN_CONTAINER task...")
	tags := map[string]string{
		shared.Tags.TASK_TYPE : shared.TaskTypes.TEST_TASK,
		shared.Tags.CONTAINER_NAME: containerID,
		shared.Tags.FILESERVER_IP: sched.ExternalServer,
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) RunContainerTask(containerName string) {
	if val, ok := sched.ContainerSlaveMap[containerName]; ok {
		msg := containerName+" has already been launched on "+val
		log.Infof(msg)
		return
	}
	log.Infoln("Generating RUN_CONTAINER task...")
	tags := map[string]string{
		shared.Tags.TASK_TYPE : shared.TaskTypes.RUN_CONTAINER,
		shared.Tags.CONTAINER_NAME: containerName,
		shared.Tags.FILESERVER_IP: sched.ExternalServer,
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) CheckpointContainerTask(containerName string) {
	if _, ok := sched.ContainerSlaveMap[containerName]; !ok {
		msg := containerName+" has not been launched yet!"
		log.Infof(msg)
		return
	}
	log.Infoln("Generating CHECKPOINT_CONTAINER task...")
	tags := map[string]string{
		shared.Tags.TASK_TYPE : shared.TaskTypes.CHECKPOINT_CONTAINER,
		shared.Tags.CONTAINER_NAME: containerName,
		shared.Tags.FILESERVER_IP: sched.ExternalServer,
		shared.Tags.TARGET_HOST: sched.ContainerSlaveMap[containerName],
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) RestoreContainerTask(containerName string, targetHost string) {
	log.Infoln("Generating RESTORE_CONTAINER task...")
	tags := map[string]string{
		shared.Tags.TASK_TYPE : shared.TaskTypes.RESTORE_CONTAINER,
		shared.Tags.CONTAINER_NAME: containerName,
		shared.Tags.FILESERVER_IP: sched.ExternalServer,
		shared.Tags.TARGET_HOST: targetHost,
	}
	task := sched.genTask(tags)
	sched.pushTask(task)
}

func (sched *ExampleScheduler) GetLogsTask(containerName string) {
	if _, ok := sched.ContainerSlaveMap[containerName]; !ok {
		msg := containerName+" has not been launched yet!"
		log.Infof(msg)
		return
	}
	log.Infoln("Generating GET_LOGS task...")
	tags := map[string]string{
		shared.Tags.TASK_TYPE : shared.TaskTypes.GET_LOGS,
		shared.Tags.CONTAINER_NAME: containerName,
		shared.Tags.FILESERVER_IP: sched.ExternalServer,
		shared.Tags.TARGET_HOST: sched.ContainerSlaveMap[containerName],
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
		log.Infoln("Tag being processed: "+key+" : "+value)
		labels.Labels = append(labels.Labels, shared.CreateLabel(key, value))
		log.Infoln("Current tags: %v", labels)
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
