package main

import (
	"encoding/base64"
	"flag"
	"fmt"

	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type scraperExecutor struct {
	tasksLaunched int
}

func newScraperExecutor() *scraperExecutor {
	return &scraperExecutor{tasksLaunched: 0}
}

func (exec *scraperExecutor) Registered(driver exec.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *scraperExecutor) Reregistered(driver exec.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *scraperExecutor) Disconnected(exec.ExecutorDriver) {
	fmt.Println("Executor disconnected.")
}

func (exec *scraperExecutor) LaunchTask(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	fmt.Printf("Launching task %v with data [%#x]\n", taskInfo.GetName(), taskInfo.Data)

	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}

	exec.tasksLaunched++
	fmt.Println("Total tasks launched ", exec.tasksLaunched)

	// Download html
	uri := string(taskInfo.Data)
	fileName, url, err := downloadHTML(uri)

	if err != nil {
		fmt.Printf("Failed to scrape html with error: %v\n", err)
		return
	}
	fmt.Printf("Scraped URI: %v\n", fileName)

	// Upload html
	path := base64.StdEncoding.EncodeToString([]byte(url))
	fmt.Printf("Uploading html: %v\n", fileName)
	if err = uploadImageToS3(path, fileName); err != nil {
		fmt.Printf("Failed to upload html with error: %v\n", err)
		return
	} else {
		fmt.Printf("Uploaded html: %v\n", fileName)
	}

	// Finish task
	fmt.Println("Finishing task", taskInfo.GetName())
	finStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
		Data:   []byte(uri),
	}
	_, err = driver.SendStatusUpdate(finStatus)
	if err != nil {
		fmt.Println("Got error", err)
		return
	}

	fmt.Println("Task finished", taskInfo.GetName())
}

func (exec *scraperExecutor) KillTask(exec.ExecutorDriver, *mesos.TaskID) {
	fmt.Println("Kill task")
}

func (exec *scraperExecutor) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	fmt.Println("Got framework message: ", msg)
}

func (exec *scraperExecutor) Shutdown(exec.ExecutorDriver) {
	fmt.Println("Shutting down the executor")
}

func (exec *scraperExecutor) Error(driver exec.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}

func init() {
	flag.Parse()
}

func main() {
	fmt.Println("Starting ScraperExecutor (Go)")

	dconfig := exec.DriverConfig{
		Executor: newScraperExecutor(),
	}
	driver, err := exec.NewMesosExecutorDriver(dconfig)

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
