package streamer

import (
	"fmt"
)

// VideoProcessingJob is the unit of work to be performed
type VideoProcessingJob struct {
	Video Video
}

// newVideoWorker takes a numeric id and a channel w/ worker pool.
func newVideoWorker(id int, workerPool chan chan VideoProcessingJob) videoWorker {
	return videoWorker{
		id:         id,
		jobQueue:   make(chan VideoProcessingJob),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

// videoWorker holds info for a pool worker.
type videoWorker struct {
	id         int
	jobQueue   chan VideoProcessingJob
	workerPool chan chan VideoProcessingJob
	quitChan   chan bool
}

// start starts the worker.
func (w videoWorker) start() {
	go func() {
		for {
			// Add jobQueue to the worker pool.
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				w.processVideoJob(job.Video)
			case <-w.quitChan:
				fmt.Printf("worker%d stopping\n", w.id)
				return
			}
		}
	}()
}

// stop the worker.
//func (w videoWorker) stop() {
//	go func() {
//		w.quitChan <- true
//	}()
//}

// VideoDispatcher holds info for a dispatcher.
type VideoDispatcher struct {
	workerPool chan chan VideoProcessingJob
	maxWorkers int
	jobQueue   chan VideoProcessingJob
}

// Run runs the workers.
func (d *VideoDispatcher) Run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := newVideoWorker(i+1, d.workerPool)
		worker.start()
	}

	go d.dispatch()
}

// dispatch dispatches a worker.
func (d *VideoDispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			go func() {
				workerJobQueue := <-d.workerPool
				workerJobQueue <- job
			}()
		}
	}
}

// processVideoJob processes the main queue job.
func (w videoWorker) processVideoJob(video Video) {
	_, _ = video.Encode()
}
