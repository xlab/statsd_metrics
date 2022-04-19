package metrics

func CustomReport(reportFn func(s Statter, tagSpec []string), tags ...Tags) {
	clientMux.RLock()
	defer clientMux.RUnlock()

	if client == nil {
		return
	}

	reportFn(client, JoinTags(tags...))
}
