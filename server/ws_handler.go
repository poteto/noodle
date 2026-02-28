package server

import (
	"net/http"
)

// handleWS upgrades an HTTP connection to WebSocket.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &wsClient{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  s.ws,
	}

	s.ws.addClient(client)

	// Send initial snapshot immediately via the send channel.
	initial, err := s.ws.loadInitialSnapshot(s.runtimeDir, s.now, s.provider, s.warnings)
	if err == nil {
		client.Send(initial)
	}

	go client.writePump()
	go client.readPump(s)
}
