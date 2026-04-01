package analyze

func ResetCachesForTests() {
	previewMu.Lock()
	previewCache = map[string]previewCacheEntry{}
	previewInFlight = map[string]*previewCall{}
	previewMu.Unlock()

	scanMu.Lock()
	scanCache = map[string]scanCacheEntry{}
	scanInFlight = map[string]*scanCall{}
	scanMu.Unlock()
}
