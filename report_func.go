package metrics

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	log "github.com/xlab/suplog"
)

func ReportFuncError(tags ...Tags) {
	fn := funcName()
	reportFunc(fn, "error", tags...)
}

func ReportClosureFuncError(name string, tags ...Tags) {
	reportFunc(name, "error", tags...)
}

func ReportFuncStatus(tags ...Tags) {
	fn := funcName()
	reportFunc(fn, "status", tags...)
}

func ReportClosureFuncStatus(name string, tags ...Tags) {
	reportFunc(name, "status", tags...)
}

func ReportFuncCall(tags ...Tags) {
	fn := funcName()
	reportFunc(fn, "called", tags...)
}

func ReportClosureFuncCall(name string, tags ...Tags) {
	reportFunc(name, "called", tags...)
}

func reportFunc(fn, action string, tags ...Tags) {
	clientMux.RLock()
	defer clientMux.RUnlock()
	if client == nil {
		return
	}

	tagArray := JoinTags(tags...)
	tagArray = append(tagArray, getSingleTag("func_name", fn))
	client.Incr(fmt.Sprintf("func.%v", action), tagArray)
}

type StopTimerFunc func()

func ReportFuncTiming(tags ...Tags) StopTimerFunc {
	clientMux.RLock()
	defer clientMux.RUnlock()
	if client == nil {
		return func() {}
	}
	t := time.Now()
	fn := funcName()

	tagArray := JoinTags(tags...)
	tagArray = append(tagArray, getSingleTag("func_name", fn))

	doneC := make(chan struct{})
	go func(name string, start time.Time) {
		timeout := time.NewTimer(config.StuckFunctionTimeout)
		defer timeout.Stop()

		select {
		case <-doneC:
			return
		case <-timeout.C:
			clientMux.RLock()
			defer clientMux.RUnlock()

			err := fmt.Errorf("detected stuck function: %s stuck for %v", name, time.Since(start))
			log.WithError(err).Warningln("detected stuck function")
			client.Incr("func.stuck", tagArray)

		}
	}(fn, t)

	return func() {
		d := time.Since(t)
		close(doneC)

		clientMux.RLock()
		defer clientMux.RUnlock()
		client.Timing("func.timing", d, tagArray)
	}
}

func ReportClosureFuncTiming(name string, tags ...Tags) StopTimerFunc {
	clientMux.RLock()
	defer clientMux.RUnlock()
	if client == nil {
		return func() {}
	}
	t := time.Now()
	tagArray := JoinTags(tags...)
	tagArray = append(tagArray, getSingleTag("func_name", name))

	doneC := make(chan struct{})
	go func(name string, start time.Time) {
		timeout := time.NewTimer(config.StuckFunctionTimeout)
		defer timeout.Stop()

		select {
		case <-doneC:
			return
		case <-timeout.C:
			clientMux.RLock()
			defer clientMux.RUnlock()

			err := fmt.Errorf("detected stuck function: %s stuck for %v", name, time.Since(start))
			log.WithError(err).Warningln("detected stuck function")
			client.Incr("func.stuck", tagArray)

		}
	}(name, t)

	return func() {
		d := time.Since(t)
		close(doneC)

		clientMux.RLock()
		defer clientMux.RUnlock()
		client.Timing("func.timing", d, tagArray)

	}
}

func funcName() string {
	pc, _, _, _ := runtime.Caller(2)
	fullName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(fullName, "/")
	nameParts := strings.Split(parts[len(parts)-1], ".")
	return nameParts[len(nameParts)-1]
}

type Tags map[string]string

func (t Tags) With(k, v string) Tags {
	if t == nil || len(t) == 0 {
		return map[string]string{
			k: v,
		}
	}

	t[k] = v
	return t
}

// JoinTags decides how to join tags based on agent used
func JoinTags(tags ...Tags) []string {
	if len(tags) == 0 {
		return []string{}
	}

	allTags := make([]string, 0, len(tags))
	for _, tagSet := range tags {
		for k, v := range tagSet {
			allTags = append(allTags, getSingleTag(k, v))
		}
	}

	return allTags
}

func getSingleTag(key, value string) string {
	if config.Agent == DatadogAgent {
		return fmt.Sprintf("%s:%s", key, value)
	}

	return fmt.Sprintf("%s=%s", key, value)
}
