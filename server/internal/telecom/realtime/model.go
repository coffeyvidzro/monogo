package realtime

import "time"

type Config struct {
	AuthSecret string
	URLs       []string
}

type ICECredentials struct {
	ICEServers []ICEServer `json:"ice_servers"`
	ExpiresAt  time.Time   `json:"expires_at"`
}

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
}
