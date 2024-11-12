package newlogd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller/elog"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
	"google.golang.org/protobuf/encoding/protojson"
)

var eveNode *tk.EveNode
var logT *testing.T
var logs chan *elog.FullLogEntry = make(chan *elog.FullLogEntry)

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
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"debug.default.loglevel":        "debug",
			"debug.default.remote.loglevel": "warning",
		},
	)

	// check logs from now on - there should be no logs with level lower than warning
	go failTestIfLogsBelowWarning()
	go eveNode.GetLogsFromAdam(getLogsBelowWarning)

	logInfof("STEP 2: do stuff to generate some logs")
	appName := tk.GetRandomAppName(projectName + "-")
	pubPorts := []string{sshPort + ":22"}
	pc := tk.GetDefaultVMConfig(appName, tk.AppDefaultCloudConfig, pubPorts)
	err := eveNode.EveDeployApp(appLink, false, pc)
	if err != nil {
		logFatalf("Failed to deploy app: %v", err)
	}

	// wait for the app to show up in the list
	time.Sleep(10 * time.Second)
	// wait 5 minutes for the app to start
	logInfof("Waiting for app %s to start...", appName)
	err = eveNode.AppWaitForRunningState(appName, appWait)
	if err != nil {
		logFatalf("Failed to wait for app to start: %v", err)
	}

	err = eveNode.AppStopAndRemove(appName)
	if err != nil {
		logInfof("Failed to stop and remove app: %v", err)
	}

	logInfof("STEP 3: check logs")
	keptLogs, err := eveNode.EveRunCommand("zcat /persist/newlog/keepSentQueue/dev.log*")
	if err != nil {
		logFatalf("Failed to get keepSentQueue logs: %v", err)
	}

	fmt.Println(keptLogs)

	// // let's see the output first
	// out, err := eveNode.EveRunCommand("stat /persist/newlog/collect/dev.log*")
	// if err != nil {
	// 	logFatalf("Failed to get devUpload logs: %v", err)
	// }
	// logInfof("%s", string(out))

	// if exists, err := eveNode.EveFileExists("/persist/newlog/collect/dev.log*"); err != nil || !exists {
	// 	logFatalf("No logs found in /persist/newlog/collect/dev.log*")
	// }

	// // TODO: replace this with EVE's logs from the controller
	// // there should be no logs with level lower than warning in devUpload
	// if exists, err := eveNode.EveFileExists("/persist/newlog/devUpload/dev.log*"); err == nil && exists {
	// 	devUploadLogs, err := eveNode.EveRunCommand("zcat /persist/newlog/devUpload/dev.log*")
	// 	if err != nil {
	// 		logFatalf("Failed to get devUpload logs: %v", err)
	// 	}
	// 	fmt.Println(devUploadLogs)
	// }
}

func failTestIfLogsBelowWarning() {
	// if anything comes out of the logs channel, fail the test
	something := <-logs

	b, err := protojson.Marshal(something)
	if err != nil {
		logFatalf("Failed to marshal log entry: %v", err)
	}
	logFatalf("Got a log with severity below warning: %s", string(b))
	logT.FailNow()
}

func getLogsBelowWarning(logEntry *elog.FullLogEntry) bool {
	if logEntry.Severity != "warning" && logEntry.Severity != "error" {
		logs <- logEntry
	}
	return false // return false to continue checking
}
