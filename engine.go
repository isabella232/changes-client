package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
    STATUS_QUEUED = "queued"
    STATUS_IN_PROGRESS = "in_progress"
    STATUS_FINISHED = "finished"
)

func reportChunks(r *Reporter, cID string, c chan LogChunk) {
	for l := range c {
		fmt.Printf("Got another chunk from %s (%d-%d)\n", l.Source, l.Offset, l.Length)
		fmt.Printf("%s", l.Payload)
		r.PushLogChunk(cID, l)
	}
}

func publishArtifacts(reporter *Reporter, cID string, artifacts []string) {
	if len(artifacts) == 0 {
		return
	}

	var matches []string
	for _, pattern := range artifacts {
		m, err := filepath.Glob(pattern)
		if err != nil {
			panic("Invalid artifact pattern" + err.Error())
		}
		matches = append(matches, m...)
	}

	reporter.PushArtifacts(cID, matches)
}

func RunCmds(reporter *Reporter, config *Config) {
    result := "passed"
    defer reporter.PushJobStatus(config.JobID, STATUS_FINISHED, result)

	wg := sync.WaitGroup{}
    reporter.PushJobStatus(config.JobID, STATUS_IN_PROGRESS, "")

	for _, cmd := range config.Cmds {
		reporter.PushStatus(cmd.Id, STATUS_IN_PROGRESS, -1)
		r, err := NewRunner(cmd.Id, cmd.Script)
		if err != nil {
			reporter.PushStatus(cmd.Id, STATUS_FINISHED, 255)
            result = "failed"
			break
		}

		env := os.Environ()
		for k, v := range cmd.Env {
			env = append(env, k+"="+v)
		}
		r.Cmd.Env = env

        if len(cmd.Cwd) > 0 {
            r.Cmd.Dir = cmd.Cwd
        }

		wg.Add(1)
		go func() {
			reportChunks(reporter, config.JobID, r.ChunkChan)
			wg.Done()
		}()

		pState, err := r.Run()
		if err != nil {
			reporter.PushStatus(cmd.Id, STATUS_FINISHED, 255)
            result = "failed"
			break
		} else {
            if pState.Success() {
                reporter.PushStatus(cmd.Id, STATUS_FINISHED, 0)
            } else {
                reporter.PushStatus(cmd.Id, STATUS_FINISHED, 1)
                result = "failed"
                break
            }
		}

		publishArtifacts(reporter, config.JobID, cmd.Artifacts)
	}

	wg.Wait()
    result = "passed"
}
