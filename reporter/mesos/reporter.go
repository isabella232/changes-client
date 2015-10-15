package mesosreporter

import (
	"log"
	"os"
	"strconv"

	"github.com/dropbox/changes-client/client"
	"github.com/dropbox/changes-client/client/adapter"
	"github.com/dropbox/changes-client/client/reporter"
)

// A reporter that connects and reports to a specific jobstep id.
// Each jobstep id has a number of endpoints associated with it that
// allows the reporter to update the status of logs, snapshots, etc.
type Reporter struct {
	reporter.DefaultReporter
}

func (r *Reporter) PushJobstepStatus(status string, result string) {
	log.Printf("[reporter] Pushing status %s", status)
	form := make(map[string]string)
	form["status"] = status
	if len(result) > 0 {
		form["result"] = result
	}

	hostname, err := os.Hostname()
	if err == nil {
		form["node"] = hostname
	}
	r.PublishChannel <- reporter.ReportPayload{Path: r.JobstepAPIPath(), Data: form, Filename: ""}
}

func (r *Reporter) PushCommandStatus(cID string, status string, retCode int) {
	form := make(map[string]string)
	form["status"] = status
	if retCode >= 0 {
		form["return_code"] = strconv.Itoa(retCode)
	}
	r.PublishChannel <- reporter.ReportPayload{Path: "/commands/" + cID + "/", Data: form, Filename: ""}
}

func (r *Reporter) PushLogChunk(source string, payload []byte) {
	form := make(map[string]string)
	form["source"] = source
	form["text"] = string(payload)
	if r.Debug {
		log.Print(string(payload))
	}
	r.PublishChannel <- reporter.ReportPayload{Path: r.JobstepAPIPath() + "logappend/", Data: form, Filename: ""}
}

func (r *Reporter) PushCommandOutput(cID string, status string, retCode int, output []byte) {
	form := make(map[string]string)
	form["status"] = status
	form["output"] = string(output)
	if retCode >= 0 {
		form["return_code"] = strconv.Itoa(retCode)
	}
	r.PublishChannel <- reporter.ReportPayload{Path: "/commands/" + cID + "/", Data: form, Filename: ""}
}

func (r *Reporter) PublishArtifacts(cmd client.ConfigCmd, a adapter.Adapter, clientLog *client.Log) error {
	if len(cmd.Artifacts) == 0 {
		clientLog.Printf("==> Skipping artifact collection")
		return nil
	}

	clientLog.Printf("==> Collecting artifacts matching %s", cmd.Artifacts)

	matches, err := a.CollectArtifacts(cmd.Artifacts, clientLog)
	if err != nil {
		clientLog.Printf("==> ERROR: %s", err)
		return err
	}

	for _, artifact := range matches {
		clientLog.Printf("==> Found: %s", artifact)
	}

	return r.pushArtifacts(matches)
}

func (r *Reporter) pushArtifacts(artifacts []string) error {
	// TODO: PushArtifacts is synchronous due to races with Adapter.Shutdown(), but
	// really what we'd want to do is just say "wait until channel empty, ok continue"
	var firstError error
	for _, artifact := range artifacts {
		e := r.SendPayload(reporter.ReportPayload{Path: r.JobstepAPIPath() + "artifacts/", Data: nil, Filename: artifact})
		if e != nil && firstError == nil {
			firstError = e
		}
	}
	return firstError
}

func New() reporter.Reporter {
	return &Reporter{}
}

func init() {
	reporter.Register("mesos", New)
}
