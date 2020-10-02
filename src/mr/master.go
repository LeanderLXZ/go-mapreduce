package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type Master struct {
	// Your definitions here.
	inputFiles []string //rest files list
	taskList   []Task   //working files--workerId list
	workerList []int

	workerNum int
	taskId    int
	mapDone   bool
	allDone   bool

	RWMutex  sync.Mutex
	RQTMutex sync.Mutex
	RPTMutex sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	DPrintf("Worker has called the Example RPC\n")
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	os.Remove("mr-socket")
	l, e := net.Listen("unix", "mr-socket")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	return m.allDone
}

//RegisterWorker is an RPC method that is called by workers after they have started
// up to report that they are ready to receive tasks.
func (m *Master) RegisterWorker(args *RegisterWorkerArgs, reply *RegisterWorkerReply) error {
	m.RWMutex.Lock()
	m.workerNum++
	reply.workerId = m.workerNum
	// reply.InputFiles = m.inputFiles
	m.RWMutex.Unlock()
	// DPrintf("Sending file list: %v\n", reply.InputFiles)
	return nil
}

//RequestTask is an RPC method that is called by workers to request a map or reduce task
func (m *Master) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	m.RQTMutex.Lock()
	if m.allDone == false {
		if m.mapDone == false { //map task
			if len(m.inputFiles) != 0 {
				m.taskId++
				time := time.Now().Unix()
				task := Task{m.taskId, m.inputFiles[0], args.workerId, time}

				reply.fileName = task.files
				reply.taskMode = "map"
				reply.taskId = task.taskId

				m.inputFiles = m.inputFiles[1:]
				//workerlist update, do I need workerlist?
				m.taskList = append(m.taskList, task)
			} else {
				// tell worker wait new task
				reply.taskMode = "wait"
			}
		} else { //reduce task
			if len(m.inputFiles) != 0 {
				m.taskId++
				time := time.Now().Unix()
				task := Task{m.taskId, m.inputFiles[0], args.workerId, time}

				reply.fileName = task.files
				reply.taskMode = "reduce"
				reply.taskId = task.taskId

				m.inputFiles = m.inputFiles[1:]
				m.taskList = append(m.taskList, task)
			} else {
				// tell worker wait new task
				reply.taskMode = "wait"
			}
		}
	} else { // Alldone
		reply.taskMode = "done"
	}

	m.RQTMutex.Unlock()
	return nil
}

func UpdateTaskList(taskList []Task, taskId int) []Task {
	for i := 0; i < len(taskList); i++ { //update taskList
		if taskList[i].taskId == taskId {
			taskList = append(taskList[:i], taskList[i+1:]...)
		}
	}
	return taskList
}

func CheckTaskList(taskList []Task, taskId int) (string, int, int64) {
	var fileName string
	var workerId int
	var time int64

	for i := 0; i < len(taskList); i++ { //update taskList
		if taskList[i].taskId == taskId {
			fileName = taskList[i].files
			workerId = taskList[i].workerId
			time = taskList[i].time
		}
	}
	return fileName, workerId, time
}

func (m *Master) UpdateTaskMode() error {
	if len(m.inputFiles) == 0 && len(m.taskList) == 0 {
		m.mapDone = true

	}
	return nil
}

//ReportTask is an RPC method that is called by workers to report a task's status
//whenever a task is finished or failed
//HINT: when a task is failed, master should reschedule it.
func (m *Master) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	m.RPTMutex.Lock()
	msg := args.msg

	if msg == "failed" {
		var fileName string
		fileName, _, _ = CheckTaskList(m.taskList, args.taskId)
		m.taskList = UpdateTaskList(m.taskList, args.taskId)
		m.inputFiles = append(m.inputFiles, fileName)
		// reply.taskMode = "wait"
	} else if msg == "done" {
		m.taskList = UpdateTaskList(m.taskList, args.taskId)
		m.UpdateTaskMode()
		// reply.taskMode = "wait"
	} else if msg == "working" {
		time1 := time.Now().Unix()
		_, _, time0 := CheckTaskList(m.taskList, args.taskId)
		if time0-time1 > 10 {
			m.taskList = UpdateTaskList(m.taskList, args.taskId)
			// reply.taskMode = "wait"
		}
	}
	m.RPTMutex.Unlock()
	return nil
}

//
// create a Master.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.
	m.inputFiles = files
	m.workerNum = 0
	m.taskId = 0
	m.mapDone = false
	m.allDone = false

	go m.server()

	return &m
}
