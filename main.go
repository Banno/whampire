package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/gogo/protobuf/proto"

	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	. "github.com/mesosphere/mesos-framework-tutorial/scheduler"
	. "github.com/mesosphere/mesos-framework-tutorial/server"
)

const (
	CPUS_PER_TASK       = 1
	MEM_PER_TASK        = 128
	defaultArtifactPort = 12345
	defaultImage        = "http://www.gabrielhartmann.com/Things/Plants/i-W2N2Rxp/0/O/DSCF6636.jpg"
)

var (
	address      = flag.String("address", "127.0.0.1", "Binding address for artifact server")
	artifactPort = flag.Int("artifactPort", defaultArtifactPort, "Binding port for artifact server")
	master       = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
	executorPath = flag.String("executor", "./example_executor", "Path to test executor")
)

func init() {
	flag.Parse()
}

func main() {
	// Start HTTP server hosting executor binary
	uri := ServeSchedulerAPI(*address, *artifactPort)

	// Executor
	executorInfo := prepareExecutorInfo(uri, getExecutorCmd(*executorPath))

	// Scheduler
	scheduler, err := NewScraperScheduler(executorInfo, CPUS_PER_TASK, MEM_PER_TASK)
	if err != nil {
		log.Fatalf("Failed to create scheduler with error: %v\n", err)
		os.Exit(-2)
	}

	// Framework
	frameworkInfo := &mesos.FrameworkInfo{
		User: proto.String(""), // Mesos-go will fill in user.
		Name: proto.String("Web Scraper"),
	}

	// Scheduler Driver
	config := sched.DriverConfig{
		Scheduler:      scheduler,
		Framework:      frameworkInfo,
		Master:         *master,
		Credential:     (*mesos.Credential)(nil),
		BindingAddress: parseIP(*address),
	}

	driver, err := sched.NewMesosSchedulerDriver(config)

	if err != nil {
		log.Fatalf("Unable to create a SchedulerDriver: %v\n", err.Error())
		os.Exit(-3)
	}

	if stat, err := driver.Run(); err != nil {
		log.Fatalf("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
		os.Exit(-4)
	}
}

func prepareExecutorInfo(uri string, cmd string) *mesos.ExecutorInfo {
	executorUris := []*mesos.CommandInfo_URI{
		{
			Value:      &uri,
			Executable: proto.Bool(true),
		},
	}

	return &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Name:       proto.String("Scraper Executor (Go)"),
		Source:     proto.String("go_test"),
		Command: &mesos.CommandInfo{
			Value: proto.String(cmd),
			Uris:  executorUris,
		},
	}
}

func getExecutorCmd(path string) string {
	return fmt.Sprintf("ACCESS_KEY=%s SECRET_KEY=%s .%s", os.Getenv("ACCESS_KEY"), os.Getenv("SECRET_KEY"), GetHttpPath(path))
}

func parseIP(address string) net.IP {
	addr, err := net.LookupIP(address)
	if err != nil {
		log.Fatal(err)
	}
	if len(addr) < 1 {
		log.Fatalf("failed to parse IP from address '%v'", address)
	}
	return addr[0]
}
