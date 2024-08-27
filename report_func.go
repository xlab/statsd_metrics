package metrics

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
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

type SafeMap struct {
	mux *sync.RWMutex
	m   map[string]string
}

func newSafeMap() SafeMap {
	return SafeMap{
		mux: new(sync.RWMutex),
		m:   make(map[string]string),
	}
}

func newSafeMapWith(k, v string) SafeMap {
	return SafeMap{
		mux: new(sync.RWMutex),
		m: map[string]string{
			k: v,
		},
	}
}

func (sm SafeMap) IsValid() bool {
	return sm.mux != nil
}

func (sm SafeMap) Set(k, v string) SafeMap {
	sm.mux.Lock()
	sm.m[k] = v
	sm.mux.Unlock()

	return sm
}

func (sm SafeMap) RLock() {
	sm.mux.RLock()
}

func (sm SafeMap) RUnlock() {
	sm.mux.RUnlock()
}

func (sm SafeMap) Map() map[string]string {
	return sm.m
}

type Tags SafeMap

func (t Tags) With(k, v string) Tags {
	if !SafeMap(t).IsValid() {
		return Tags(newSafeMapWith(k, v))
	}

	return Tags(SafeMap(t).Set(k, v))
}

// WithBaseTags allows to inject metrics BaseTags into custom tag set. Useful when
// tags are used outside of StatsD reporting. For example, within InfluxDB points.
func (t Tags) WithBaseTags() Tags {
	baseTags := config.BaseTagsMap()
	allTags := newSafeMap()

	for k, v := range baseTags {
		allTags.m[k] = v
	}

	for k, v := range t.m {
		allTags.m[k] = v
	}

	return Tags(allTags)
}

// NewTags unions unsafe maps as a safe maps used for tags.
func NewTags(mapsToUnion ...map[string]string) Tags {
	if len(mapsToUnion) == 0 {
		return Tags(newSafeMap())
	}

	safeMap := SafeMap{
		mux: new(sync.RWMutex),
		m:   make(map[string]string, len(mapsToUnion[0])),
	}

	for _, m := range mapsToUnion {
		for k, v := range m {
			safeMap.m[k] = v
		}
	}

	return Tags(safeMap)
}

// JoinTags decides how to join tags based on agent used
func JoinTags(tags ...Tags) []string {
	if len(tags) == 0 {
		return []string{}
	}

	allTags := make([]string, 0, len(tags))
	for _, tagSet := range tags {
		safeMap := SafeMap(tagSet)
		if !safeMap.IsValid() {
			continue
		}

		safeMap.RLock()

		m := safeMap.Map()
		for k, v := range m {
			allTags = append(allTags, getSingleTag(k, v))
		}

		safeMap.RUnlock()
	}

	return allTags
}

func getSingleTag(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
