package mapreduce

import (
	"log"
	"sync"
)

// Schedules map operations on remote workers. This will run until InputFilePathChan
// is closed. If there is no worker available, it'll block.
func (master *Master) schedule(task *Task, proc string, filePathChan chan string) int {
	var (
		wg        sync.WaitGroup
		filePath  string
		worker    *RemoteWorker
		operation *Operation
		counter   int
	)

	log.Printf("Scheduling %v operations\n", proc)

	counter = 0
	for filePath = range filePathChan {
		operation = &Operation{proc, counter, filePath}
		counter++

		worker = <-master.idleWorkerChan
		wg.Add(1)
		go master.runOperation(worker, operation, &wg)
	}

	wg.Wait()

	// Retry failed operations only if there are any
	failedCounter := 0
RetryLoop:
	for {
		select {
		case failedOp := <-master.failedOperationChan:
			log.Printf("Re-executing failed operation %v", failedOp.id)
			worker = <-master.idleWorkerChan
			wg.Add(1)
			go master.runOperation(worker, failedOp, &wg)
			failedCounter++
		default:
			log.Printf("Waiting for failed operations to complete")
			wg.Wait()

			if failedCounter == 0 {
				log.Printf("All failed operations retried")
				break RetryLoop
			}
			failedCounter = 0
		}
	}

	log.Printf("%vx %v operations completed\n", counter, proc)
	return counter
}

// runOperation start a single operation on a RemoteWorker and wait for it to return or fail.
func (master *Master) runOperation(remoteWorker *RemoteWorker, operation *Operation, wg *sync.WaitGroup) {
	var (
		err  error
		args *RunArgs
	)

	log.Printf("Running %v (ID: '%v' File: '%v' Worker: '%v')\n", operation.proc, operation.id, operation.filePath, remoteWorker.id)

	args = &RunArgs{operation.id, operation.filePath}
	err = remoteWorker.callRemoteWorker(operation.proc, args, new(struct{}))

	if err != nil {
		log.Printf("Operation %v '%v' Failed. Error: %v\n", operation.proc, operation.id, err)
		wg.Done()
		master.failedWorkerChan <- remoteWorker
		master.failedOperationChan <- operation
	} else {
		wg.Done()
		master.idleWorkerChan <- remoteWorker
	}
}
