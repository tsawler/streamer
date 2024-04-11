package streamer

// VideoProcessingJob is the unit of work to be performed. We wrap this type
// around a Video, which has all the information we need about the input source
// and what we want the output to look like.
type VideoProcessingJob struct {
	Video Video
}

// newVideoWorker takes a numeric id and a channel which accepts the chan VideoProcessingJob
// type, and returns a videoWorker object.
func newVideoWorker(id int, workerPool chan chan VideoProcessingJob) videoWorker {
	return videoWorker{
		id:         id,
		jobQueue:   make(chan VideoProcessingJob),
		workerPool: workerPool,
	}
}

// videoWorker holds info for a pool worker. It has the numeric id of the worker,
// the job queue, and the worker pool chan. A chan chan is used when the thing you want to
// send down a channel is another channel to send things back.
// See http://tleyden.github.io/blog/2013/11/23/understanding-chan-chans-in-go/
type videoWorker struct {
	id         int
	jobQueue   chan VideoProcessingJob      // Where we send jobs to process.
	workerPool chan chan VideoProcessingJob // Our worker pool channel.
}

// start starts a worker.
func (w videoWorker) start() {
	go func() {
		for {
			// Add jobQueue to the worker pool.
			w.workerPool <- w.jobQueue

			// Wait for a job to come back.
			job := <-w.jobQueue

			// Process the video with a worker.
			w.processVideoJob(job.Video)
		}
	}()
}

// VideoDispatcher holds info for a dispatcher.
type VideoDispatcher struct {
	WorkerPool chan chan VideoProcessingJob // Our worker pool channel.
	maxWorkers int                          // The maximum number of workers in our pool.
	jobQueue   chan VideoProcessingJob      // The channel we send work to.
	Processor  Processor
}

// Run runs the workers.
func (vd *VideoDispatcher) Run() {
	for i := 0; i < vd.maxWorkers; i++ {
		worker := newVideoWorker(i+1, vd.WorkerPool)
		worker.start()
	}

	go vd.dispatch()
}

// dispatch dispatches a worker.
func (vd *VideoDispatcher) dispatch() {
	for {
		// Wait for a job to come in.
		job := <-vd.jobQueue

		go func() {
			workerJobQueue := <-vd.WorkerPool // assign a channel from our worker pool to workerJobPool.
			workerJobQueue <- job             // Send the unit of work to our queue.
		}()
	}
}

// processVideoJob processes the main queue job.
func (w videoWorker) processVideoJob(video Video) {
	video.encode()
}
