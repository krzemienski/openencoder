package worker

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/alfg/enc/api/types"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// Context defines the job context to be passed to the worker.
type Context struct {
	GUID        string
	Profile     string
	Source      string
	Destination string
}

// Log worker middleware for logging job.
func (c *Context) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	log.Infof("worker: starting job %s\n", job.Name)
	return next()
}

// FindJob worker middleware for setting job context from job arguments.
func (c *Context) FindJob(job *work.Job, next work.NextMiddlewareFunc) error {
	if _, ok := job.Args["guid"]; ok {
		c.GUID = job.ArgString("guid")
		if err := job.ArgError(); err != nil {
			return err
		}
	}
	if _, ok := job.Args["profile"]; ok {
		c.Profile = job.ArgString("profile")
		if err := job.ArgError(); err != nil {
			return err
		}
	}
	if _, ok := job.Args["source"]; ok {
		c.Source = job.ArgString("source")
		if err := job.ArgError(); err != nil {
			return err
		}
	}
	if _, ok := job.Args["destination"]; ok {
		c.Destination = job.ArgString("destination")
		if err := job.ArgError(); err != nil {
			return err
		}
	}
	return next()
}

// SendJob worker handler for running job.
func (c *Context) SendJob(job *work.Job) error {
	guid := job.ArgString("guid")
	profile := job.ArgString("profile")
	source := job.ArgString("source")
	destination := job.ArgString("destination")

	j := types.Job{
		GUID:        guid,
		Profile:     profile,
		Source:      source,
		Destination: destination,
	}

	// Start job.
	runEncodeJob(j)
	log.Infof("worker: completed %s!\n", j.Profile)
	return nil
}

func startJob(id int, j types.Job) {
	log.Infof("worker: started %s\n", j.Profile)

	// runWorkflow(j)
	log.Infof("worker: completed %s!\n", j.Profile)
}

// func (c *Context) Export(job *work.Job) error {
// 	return nil
// }

// Config defines configuration for creating a NewWorker.
type Config struct {
	Host        string
	Port        int
	Namespace   string
	JobName     string
	Concurrency uint
}

// NewWorker creates a new worker instance to listen and process jobs in the queue.
func NewWorker(workerCfg Config) {

	// Make a redis pool
	redisPool := &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp",
				fmt.Sprintf("%s:%d", workerCfg.Host, workerCfg.Port))
		},
	}

	// Make a new pool.
	pool := work.NewWorkerPool(Context{},
		workerCfg.Concurrency, workerCfg.Namespace, redisPool)

	// Add middleware that will be executed for each job
	pool.Middleware((*Context).Log)
	pool.Middleware((*Context).FindJob)

	// Map the name of jobs to handler functions
	pool.Job("encode", (*Context).SendJob)

	// Customize options:
	// pool.JobWithOptions("export", work.JobOptions{Priority: 10, MaxFails: 1}, (*Context).Export)

	// Start processing jobs
	pool.Start()

	// Wait for a signal to quit:
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	<-signalChan

	// Stop the pool
	pool.Stop()
}