package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type procState struct {
	cmd     string
	user    string
	ppid    string
	pid     string
	vsz     int64
	rss     int64
	pcpu    float64
	thcount int64
	state   string
	esec    int64
	csec    int64
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	if len(argv) != 1 {
		log.Println("please specify process pattern")
		return 1
	}
	re, err := regexp.Compile(argv[0])
	if err != nil {
		log.Printf("invalid regexp pattern: %s", err.Error())
		return 1
	}

	pidStr := strconv.Itoa(os.Getpid())
	t := time.NewTicker(5 * time.Second)
	for {
		<-t.C
		procs, _ := getProcs()
		for _, proc := range procs {
			if proc.pid == pidStr || proc.ppid == pidStr {
				continue
			}
			if re.MatchString(proc.cmd) {
				exec.Command("kill", proc.pid).Run()
				fmt.Printf("killed %s", proc.cmd)
			}
		}
	}
	return 0
}

var threadsUnknown = runtime.GOOS == "darwin"

func getProcs() (proc []procState, err error) {
	var procs []procState
	psformat := "user,ppid,pid,vsz,rss,pcpu,nlwp,state,etime,time,command"
	if threadsUnknown {
		psformat = "user,ppid,pid,vsz,rss,pcpu,state,etime,time,command"
	}
	output, _ := exec.Command("ps", "axwwo", psformat).Output()
	for _, line := range strings.Split(string(output), "\n")[1:] {
		proc, err := parseProcState(line)
		if err != nil {
			continue
		}
		procs = append(procs, proc)
	}
	return procs, nil
}

func parseProcState(line string) (proc procState, err error) {
	fields := strings.Fields(line)
	fieldsMinLen := 11
	if threadsUnknown {
		fieldsMinLen = 10
	}
	if len(fields) < fieldsMinLen {
		return procState{}, fmt.Errorf("parseProcState: insufficient words")
	}
	vsz, _ := strconv.ParseInt(fields[3], 10, 64)
	rss, _ := strconv.ParseInt(fields[4], 10, 64)
	pcpu, _ := strconv.ParseFloat(fields[5], 64)
	if threadsUnknown {
		esec := timeStrToSeconds(fields[7])
		csec := timeStrToSeconds(fields[8])
		return procState{strings.Join(fields[9:], " "), fields[0], fields[1], fields[2], vsz, rss, pcpu, 1, fields[6], esec, csec}, nil
	}
	thcount, _ := strconv.ParseInt(fields[6], 10, 64)
	esec := timeStrToSeconds(fields[8])
	csec := timeStrToSeconds(fields[9])
	return procState{strings.Join(fields[10:], " "), fields[0], fields[1], fields[2], vsz, rss, pcpu, thcount, fields[7], esec, csec}, nil
}

var timeRegexp = regexp.MustCompile(`(?:(\d+)-)?(?:(\d+):)?(\d+)[:.](\d+)`)

func timeStrToSeconds(etime string) int64 {
	match := timeRegexp.FindStringSubmatch(etime)
	if match == nil || len(match) != 5 {
		return 0
	}
	days, _ := strconv.ParseInt(match[1], 10, 64)
	hours, _ := strconv.ParseInt(match[2], 10, 64)
	minutes, _ := strconv.ParseInt(match[3], 10, 64)
	seconds, _ := strconv.ParseInt(match[4], 10, 64)
	return (((days*24+hours)*60+minutes)*60 + seconds)
}
