package scheduler

import (
	"encoding/base64"
	"github.com/gogo/protobuf/proto"
	"strconv"
	"time"

	"database/sql"
	log "github.com/golang/glog"
	_ "github.com/mattn/go-sqlite3"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

type ScraperScheduler struct {
	executor      *mesos.ExecutorInfo
	tasksLaunched int
	tasksFinished int
	totalTasks    int
	urls          []string
	cpuPerTask    float64
	memPerTask    float64
}

func NewScraperScheduler(exec *mesos.ExecutorInfo, cpuPerTask float64, memPerTask float64) (*ScraperScheduler, error) {
	urls, err := readLines("urls")
	if err != nil {
		log.Errorf("Failed to read image list with error: %v\n", err)
		return nil, err
	}

	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal("Failed to connect to database: %v\n", err)
	}
	defer db.Close()

	taskUrls := []string{}
	for _, uri := range urls {
		// query sqlite
		var _uri string
		err := db.QueryRow("SELECT uri FROM runs WHERE uri=?", uri).Scan(&_uri)
		switch {
		case err == sql.ErrNoRows:
			log.Info("No URI, starting task")
			taskUrls = append(taskUrls, uri)
		case err != nil:
			log.Fatal(err)
		default:
			log.Infof("URI already exists %s\n", _uri)
			continue
		}
	}

	return &ScraperScheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    len(taskUrls),
		urls:          taskUrls,
		cpuPerTask:    cpuPerTask,
		memPerTask:    memPerTask,
	}, nil
}

func (sched *ScraperScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Scheduler Registered with Master ", masterInfo)
}

func (sched *ScraperScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Scheduler Re-Registered with Master ", masterInfo)
}

func (sched *ScraperScheduler) Disconnected(sched.SchedulerDriver) {
	log.Infoln("Scheduler Disconnected")
}

func (sched *ScraperScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	logOffers(offers)

	for _, offer := range offers {
		if sched.tasksLaunched >= sched.totalTasks || len(sched.urls) == 0 {
			log.Infof("Declining offer %s", offer.Id.GetValue())
			driver.DeclineOffer(offer.Id, &mesos.Filters{})
			continue
		}
		remainingCpus := getOfferCpu(offer)
		remainingMems := getOfferMem(offer)

		var tasks []*mesos.TaskInfo
		for sched.cpuPerTask <= remainingCpus &&
			sched.memPerTask <= remainingMems &&
			sched.tasksLaunched < sched.totalTasks {

			log.Infof("Processing url %v of %v\n", sched.tasksLaunched, sched.totalTasks)
			log.Infof("Total Tasks: %d", sched.totalTasks)
			log.Infof("Tasks Launched: %d", sched.tasksLaunched)
			uri := sched.urls[sched.tasksLaunched]
			log.Infof("URI: %s", uri)

			sched.tasksLaunched++

			taskId := &mesos.TaskID{
				Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
			}

			task := &mesos.TaskInfo{
				Name:     proto.String("go-task-" + taskId.GetValue()),
				TaskId:   taskId,
				SlaveId:  offer.SlaveId,
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", sched.cpuPerTask),
					util.NewScalarResource("mem", sched.memPerTask),
				},
				Data: []byte(uri),
			}
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			remainingCpus -= sched.cpuPerTask
			remainingMems -= sched.memPerTask
		}
		log.Infoln("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *ScraperScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())

	if status.GetState() == mesos.TaskState_TASK_FINISHED {
		sched.tasksFinished++
		log.Infof("%v of %v tasks finished.", sched.tasksFinished, sched.totalTasks)
		uri := string(status.Data)

		db, err := sql.Open("sqlite3", "./database.db")
		if err != nil {
			log.Fatal("Failed to connect to database: %v\n", err)
		}
		defer db.Close()
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("insert into runs(uri, storage_path, last_scrape_time) values(?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}

		path := base64.StdEncoding.EncodeToString([]byte(uri))
		_, err = stmt.Exec(uri, path, time.Now().Unix())
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		tx.Commit()
	}

	if sched.tasksFinished >= sched.totalTasks {
		log.Infoln("Tasks that we know about are done!")
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		log.Infoln(
			"Aborting because task", status.TaskId.GetValue(),
			"is in unexpected state", status.State.String(),
			"with message", status.GetMessage(),
		)
		driver.Abort()
	}
}

func (sched *ScraperScheduler) OfferRescinded(s sched.SchedulerDriver, id *mesos.OfferID) {
	log.Infof("Offer '%v' rescinded.\n", *id)
}

func (sched *ScraperScheduler) FrameworkMessage(s sched.SchedulerDriver, exId *mesos.ExecutorID, slvId *mesos.SlaveID, msg string) {
	log.Infof("Received framework message from executor '%v' on slave '%v': %s.\n", *exId, *slvId, msg)
}

func (sched *ScraperScheduler) SlaveLost(s sched.SchedulerDriver, id *mesos.SlaveID) {
	log.Infof("Slave '%v' lost.\n", *id)
}

func (sched *ScraperScheduler) ExecutorLost(s sched.SchedulerDriver, exId *mesos.ExecutorID, slvId *mesos.SlaveID, i int) {
	log.Infof("Executor '%v' lost on slave '%v' with exit code: %v.\n", *exId, *slvId, i)
}

func (sched *ScraperScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}
