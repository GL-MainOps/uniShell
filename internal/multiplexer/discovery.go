package multiplexer

type DiscoveryState string

const (
	DiscoveryMissing DiscoveryState = "missing"
	DiscoveryLive    DiscoveryState = "live"
)

type DiscoveryResult struct {
	State   DiscoveryState
	Session *ManagedSession
}

func (m *Manager) DiscoverState(
	runtimePath string,
	sessionName string,
) (DiscoveryResult, error) {
	session, err := m.Discover(runtimePath, sessionName)
	if err == nil {
		return DiscoveryResult{
			State:   DiscoveryLive,
			Session: session,
		}, nil
	}

	if err == ErrSessionNotFound {
		return DiscoveryResult{
			State: DiscoveryMissing,
		}, nil
	}

	return DiscoveryResult{}, err
}
