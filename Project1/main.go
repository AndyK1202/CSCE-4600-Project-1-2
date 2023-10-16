package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sort"

	"github.com/olekukonko/tablewriter"
)

func main() {
	// CLI args
	f, closeFile, err := openProcessingFile(os.Args...)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile()

	// Load and parse processes
	processes, err := loadProcesses(f)
	if err != nil {
		log.Fatal(err)
	}

	// First-come, first-serve scheduling
	FCFSSchedule(os.Stdout, "First-come, first-serve", processes)

	//SJFSchedule(os.Stdout, "Shortest-job-first", processes)
	//
	SJFSchedule(os.Stdout, "Shortest-job-first", processes)

	//SJFPrioritySchedule(os.Stdout, "Priority", processes)
	//
	SJFPrioritySchedule(os.Stdout, "Shortest-job-first with Priority", processes)
	//RRSchedule(os.Stdout, "Round-robin", processes)
	RRSchedule(os.Stdout, "Round-robin", processes)



	

}

func openProcessingFile(args ...string) (*os.File, func(), error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%w: must give a scheduling file to process", ErrInvalidArgs)
	}
	// Read in CSV process CSV file
	f, err := os.Open(args[1])
	if err != nil {
		return nil, nil, fmt.Errorf("%v: error opening scheduling file", err)
	}
	closeFn := func() {
		if err := f.Close(); err != nil {
			log.Fatalf("%v: error closing scheduling file", err)
		}
	}

	return f, closeFn, nil
}

type (
	Process struct {
		ProcessID     int64
		ArrivalTime   int64
		BurstDuration int64
		Priority      int64
		Exit int64
		Turnaround int64
		Wait int64
	}
	TimeSlice struct {
		PID   int64
		Start int64
		Stop  int64
	}
	Gantt struct {
		PID   int64
		Start int64
		Stop  int64
	}
)

//region Schedulers

// FCFSSchedule outputs a schedule of processes in a GANTT chart and a table of timing given:
// • an output writer
// • a title for the chart
// • a slice of processes
func FCFSSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)
	for i := range processes {
		if processes[i].ArrivalTime > 0 {
			waitingTime = serviceTime - processes[i].ArrivalTime
		}
		totalWait += float64(waitingTime)

		start := waitingTime + processes[i].ArrivalTime

		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		completion := processes[i].BurstDuration + processes[i].ArrivalTime + waitingTime
		lastCompletion = float64(completion)

		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}
		serviceTime += processes[i].BurstDuration

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := totalWait / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

//func SJFPrioritySchedule(w io.Writer, title string, processes []Process) { }
//
func SJFPrioritySchedule(w io.Writer, title string, processes []Process) {
    var currentTime int64
    var completedProcesses int = 0
    var isRunning bool = false
    var currentProcess Process
    var remainingTime int64
    var queue []Process
    var gantt []TimeSlice
    var totalWaitingTime int64
    var totalTurnaroundTime int64

    // Create deep copy of processes to calculate waiting and turnaround times later
    originalProcesses := make([]Process, len(processes))
    copy(originalProcesses, processes)

    // Main loop runs until all processes are executed
    for completedProcesses < len(processes) {
        for _, p := range processes {
            if p.ArrivalTime == currentTime {
                queue = append(queue, p)
            }
        }

        if isRunning {
            if remainingTime == 0 {
                isRunning = false
                completedProcesses++
                for i, op := range originalProcesses {
                    if op.ProcessID == currentProcess.ProcessID {
                        originalProcesses[i].Exit = currentTime
                        break
                    }
                }
                gantt = append(gantt, TimeSlice{PID: currentProcess.ProcessID, Start: currentTime - currentProcess.BurstDuration, Stop: currentTime})
            }
        }

        // If no process is currently running, pick the shortest process with highest priority
        if !isRunning && len(queue) != 0 {
            // Sort the queue based on burst time and then priority
            sort.SliceStable(queue, func(i, j int) bool {
                if queue[i].BurstDuration == queue[j].BurstDuration {
                    return queue[i].Priority < queue[j].Priority
                }
                return queue[i].BurstDuration < queue[j].BurstDuration
            })
            currentProcess = queue[0]
            remainingTime = currentProcess.BurstDuration
            isRunning = true
            queue = queue[1:] // Dequeue the current process
        } else if isRunning {
            remainingTime--
        }

        currentTime++
    }

    // Calculate waiting and turnaround times
    for _, p := range originalProcesses {
        turnaroundTime := p.Exit - p.ArrivalTime
        waitingTime := turnaroundTime - p.BurstDuration
        totalWaitingTime += waitingTime
        totalTurnaroundTime += turnaroundTime
    }

    aveWait := float64(totalWaitingTime) / float64(len(processes))
    aveTurnaround := float64(totalTurnaroundTime) / float64(len(processes))
    aveThroughput := float64(len(processes)) / float64(currentTime)

    // Convert originalProcesses to [][]string format for output
    rows := make([][]string, len(originalProcesses))
    for i, p := range originalProcesses {
        rows[i] = []string{
            fmt.Sprint(p.ProcessID),
            fmt.Sprint(p.Priority),
            fmt.Sprint(p.BurstDuration),
            fmt.Sprint(p.ArrivalTime),
            fmt.Sprint(p.Exit - p.ArrivalTime - p.BurstDuration), // Wait time
            fmt.Sprint(p.Exit - p.ArrivalTime),                   // Turnaround time
            fmt.Sprint(p.Exit),
        }
    }

    // Output results
    outputTitle(w, title)
    outputGantt(w, gantt)
    outputSchedule(w, rows, aveWait, aveTurnaround, aveThroughput)
}




//func SJFSchedule(w io.Writer, title string, processes []Process) { }
//
func SJFSchedule(w io.Writer, title string, processes []Process) {
    sort.SliceStable(processes, func(i, j int) bool {
        return processes[i].BurstDuration < processes[j].BurstDuration
    })

    var (
        currentTime      int64
        totalWait        float64
        totalTurnaround  float64
        gantt            = make([]TimeSlice, 0)
        schedule         = make([][]string, len(processes))
    )

    for i := range processes {
        waitingTime := currentTime - processes[i].ArrivalTime
        if waitingTime < 0 {
            waitingTime = 0
            currentTime = processes[i].ArrivalTime
        }
        totalWait += float64(waitingTime)

        start := currentTime

        turnaround := processes[i].BurstDuration + waitingTime
        totalTurnaround += float64(turnaround)

        completion := currentTime + processes[i].BurstDuration

        schedule[i] = []string{
            fmt.Sprint(processes[i].ProcessID),
            fmt.Sprint(processes[i].Priority),
            fmt.Sprint(processes[i].BurstDuration),
            fmt.Sprint(processes[i].ArrivalTime),
            fmt.Sprint(waitingTime),
            fmt.Sprint(turnaround),
            fmt.Sprint(completion),
        }

        gantt = append(gantt, TimeSlice{
            PID:   processes[i].ProcessID,
            Start: start,
            Stop:  completion,
        })

        currentTime = completion
    }

    count := float64(len(processes))
    aveWait := totalWait / count
    aveTurnaround := totalTurnaround / count
    aveThroughput := count / float64(currentTime)

    outputTitle(w, title)
    outputGantt(w, gantt)
    outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}


//func RRSchedule(w io.Writer, title string, processes []Process) { }
func RRSchedule(w io.Writer, title string, processes []Process) {
    var clock int64 = 0
    quantum := int64(4) // Making quantum an int64 for consistent types
    var completed int64 = 0

    // Sort processes based on ArrivalTime
    sort.SliceStable(processes, func(i, j int) bool {
        return processes[i].ArrivalTime < processes[j].ArrivalTime
    })

    var queue []Process
    var gantt []TimeSlice
    var schedule []Process
    originalProcesses := make(map[int64]Process)

    for _, p := range processes {
        originalProcesses[p.ProcessID] = p
    }

    for len(processes) > 0 || len(queue) > 0 {
        //Monitor fmt.Println("Inside main loop")
        //fmt.Printf("Processes length: %d, Queue length: %d\n", len(processes), len(queue))

        for len(processes) > 0 && processes[0].ArrivalTime <= clock {
            queue = append(queue, processes[0])
            processes = processes[1:]
        }

        //Just to monitor fmt.Println("After moving to queue:", "Processes length:", len(processes), "Queue length:", len(queue))

        if len(queue) == 0 {
            clock++
            continue
        }

        currentProcess := queue[0]
        queue = queue[1:]

        timeSlice := TimeSlice{
            PID:   currentProcess.ProcessID,
            Start: clock,
        }

        if currentProcess.BurstDuration > quantum {
            timeSlice.Stop = clock + quantum
            clock += quantum
            currentProcess.BurstDuration -= quantum
            queue = append(queue, currentProcess)
        } else {
            timeSlice.Stop = clock + currentProcess.BurstDuration
            clock += currentProcess.BurstDuration
            original := originalProcesses[currentProcess.ProcessID]
            currentProcess.Exit = clock
            currentProcess.Turnaround = clock - original.ArrivalTime
            currentProcess.Wait = currentProcess.Turnaround - original.BurstDuration
            schedule = append(schedule, currentProcess)
            completed++
        }

        gantt = append(gantt, timeSlice)
       //fmt.Println("After processing process:", "Processes length:", len(processes), "Queue length:", len(queue))
    }

    // Compute averages
    var totalWait, totalTurnaround int64 = 0, 0
    for _, proc := range schedule {
        totalWait += proc.Wait
        totalTurnaround += proc.Turnaround
    }

    aveWait := float64(totalWait) / float64(completed)
    aveTurnaround := float64(totalTurnaround) / float64(completed)
    aveThroughput := float64(completed) / float64(clock)

    // Convert schedule to [][]string format for output
    rows := make([][]string, 0, len(schedule))
    for _, proc := range schedule {
        row := []string{
            strconv.FormatInt(proc.ProcessID, 10),
            strconv.FormatInt(proc.Priority, 10),
            strconv.FormatInt(originalProcesses[proc.ProcessID].BurstDuration, 10), // Original burst time
            strconv.FormatInt(proc.ArrivalTime, 10),
            strconv.FormatInt(proc.Wait, 10),
            strconv.FormatInt(proc.Turnaround, 10),
            strconv.FormatInt(proc.Exit, 10),
        }
        rows = append(rows, row)
    }

    // Output
    outputTitle(w, title)
    outputGantt(w, gantt)
    outputSchedule(w, rows, aveWait, aveTurnaround, aveThroughput)
}

//endregion

//region Output helpers

func outputTitle(w io.Writer, title string) {
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
	_, _ = fmt.Fprintln(w, strings.Repeat(" ", len(title)/2), title)
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
}

func outputGantt(w io.Writer, gantt []TimeSlice) {
	_, _ = fmt.Fprintln(w, "Gantt schedule")
	_, _ = fmt.Fprint(w, "|")
	for i := range gantt {
		pid := fmt.Sprint(gantt[i].PID)
		padding := strings.Repeat(" ", (8-len(pid))/2)
		_, _ = fmt.Fprint(w, padding, pid, padding, "|")
	}
	_, _ = fmt.Fprintln(w)
	for i := range gantt {
		_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Start), "\t")
		if len(gantt)-1 == i {
			_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Stop))
		}
	}
	_, _ = fmt.Fprintf(w, "\n\n")
}

func outputSchedule(w io.Writer, rows [][]string, wait, turnaround, throughput float64) {
	_, _ = fmt.Fprintln(w, "Schedule table")
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"ID", "Priority", "Burst", "Arrival", "Wait", "Turnaround", "Exit"})
	table.AppendBulk(rows)
	table.SetFooter([]string{"", "", "", "",
		fmt.Sprintf("Average\n%.2f", wait),
		fmt.Sprintf("Average\n%.2f", turnaround),
		fmt.Sprintf("Throughput\n%.2f/t", throughput)})
	table.Render()
}

//endregion

//region Loading processes.

var ErrInvalidArgs = errors.New("invalid args")

func loadProcesses(r io.Reader) ([]Process, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: reading CSV", err)
	}

	processes := make([]Process, len(rows))
	for i := range rows {
		processes[i].ProcessID = mustStrToInt(rows[i][0])
		processes[i].BurstDuration = mustStrToInt(rows[i][1])
		processes[i].ArrivalTime = mustStrToInt(rows[i][2])
		if len(rows[i]) == 4 {
			processes[i].Priority = mustStrToInt(rows[i][3])
		}
	}

	return processes, nil
}

func mustStrToInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return i
}

//endregion