package streamer

import (
	"fmt"
)

// VideoProcessingJob is the unit of work to be performed. We wrap this type
// around a Video, which has all the information we need about the input source
// and what we want the output to look like.
type VideoProcessingJob struct {
	Video Video
}

// newVideoWorker takes a numeric id and a channel w/ worker pool, and returns
// a videoWorker object.
func newVideoWorker(id int, workerPool chan chan VideoProcessingJob) videoWorker {
	return videoWorker{
		id:         id,
		jobQueue:   make(chan VideoProcessingJob),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

// videoWorker holds info for a pool worker. It has the numeric id of the worker,
// the job queue, and the worker pool chan. A chan chan is used when the thing you want to send down the channel is
// another channel to send things back.
// See http://tleyden.github.io/blog/2013/11/23/understanding-chan-chans-in-go/
type videoWorker struct {
	id         int
	jobQueue   chan VideoProcessingJob      // Where we send jobs to process.
	workerPool chan chan VideoProcessingJob // Our worker pool channel.
	quitChan   chan bool                    // A channel used to quit things.
}

// start starts a worker.
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
func (w videoWorker) stop() {
	go func() {
		w.quitChan <- true
	}()
}

// VideoDispatcher holds info for a dispatcher.
type VideoDispatcher struct {
	workerPool chan chan VideoProcessingJob // Our worker pool channel.
	maxWorkers int                          // The maximum number of workers in our pool.
	jobQueue   chan VideoProcessingJob      // The channel we send work to.
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
				workerJobQueue := <-d.workerPool // assign a channel from our worker pool to workerJobPool.
				workerJobQueue <- job            // Send the unit of work to our queue.
			}()
		}
	}
}

// processVideoJob processes the main queue job.
func (w videoWorker) processVideoJob(video Video) {
	_, _ = video.Encode()
}
