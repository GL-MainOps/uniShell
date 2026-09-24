package app

// ListSessions returns sessions visible to the current launch mode. Persistent
// installations include sessions from all installed runtime versions.
func (a *App) ListSessions() ([]*CleanSession, error) {
	if a.Persistent {
		return a.DiscoverPersistentCleanSessions()
	}
	return a.DiscoverCleanSessions()
}
