package newlogd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller/elog"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
	pillartypes "github.com/lf-edge/eve/pkg/pillar/types"
	"google.golang.org/protobuf/encoding/protojson"
)

var eveNode *tk.EveNode
var logT *testing.T
var logs chan *elog.FullLogEntry = make(chan *elog.FullLogEntry)
var foundLogs map[string][]*elog.FullLogEntry = make(map[string][]*elog.FullLogEntry)

const (
	projectName = "newlogd"
	sshPort     = "8027"
	appLink     = "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
	appWait     = 60 * 30
	sshWait     = 60 * 15
)

func logFatalf(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Fatal(out)
	} else {
		fmt.Print(out)
		os.Exit(1)
	}
}

func logInfof(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Logf(out)
	} else {
		fmt.Print(out)
	}
}

func TestMain(m *testing.M) {
	logInfof("newlogd Test started")
	defer logInfof("newlogd Test finished")

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		logFatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestLogLevelsDifferent(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t

	logInfof("TestLogLevelsDifferent started")
	defer logInfof("TestLogLevelsDifferent finished")

	logInfof("STEP 1: set log levels")
	desiredLogLevel := "none"
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"debug.default.loglevel":        "debug",
			"debug.syslog.loglevel":         "info",
			"debug.kernel.loglevel":         "info",
			"debug.default.remote.loglevel": desiredLogLevel,
			"debug.local.loglevel":          "all",
		},
	)

	logInfof("STEP 2: wait for the log levels to be applied")
	// TODO: change this to use the metric of when the last config was applied / changed
	time.Sleep(30 * time.Second)

	logInfof("STEP 3: start capturing logs")
	go func() {
		if err := eveNode.GetLogsFromAdam(categorizeLogs, 0); err != nil {
			logFatalf("Failed to get logs from adam: %v", err)
		}
	}()
	logInfof("STEP 4: wait for the routine to gather some logs")
	time.Sleep(60 * time.Second)

	logInfof("STEP 5: check the logs")
	fail := false
	f, err := os.OpenFile("unexpected_logs.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logFatalf("Failed to open file: %v", err)
	}
	defer f.Close()
	for severity, logs := range foundLogs {
		logInfof("Logs with severity %s: %d", severity, len(logs))
		if pillartypes.SyslogKernelLogLevelNum[severity] > pillartypes.SyslogKernelLogLevelNum[desiredLogLevel] {
			fail = true
			for _, log := range logs {
				b, err := protojson.Marshal(log)
				if err != nil {
					logFatalf("Failed to marshal log: %v", err)
				}
				if _, err := f.WriteString(string(b) + "\n"); err != nil {
					logFatalf("Failed to write to file: %v", err)
				}
			}
		}
	}

	if fail {
		logFatalf("Logs with unexpected severity found")
	}

	// TODO: reset log levels
}

func categorizeLogs(logEntry *elog.FullLogEntry) bool {
	foundLogs[logEntry.GetSeverity()] = append(foundLogs[logEntry.GetSeverity()], logEntry)
	return false // return false to continue checking
}
