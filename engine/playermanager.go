package engine

import (
	"fmt"
)

// PlayerManager XXX
type PlayerManager struct {
	players map[string]*Player
}

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players: map[string]*Player{},
	}
}

func (pm *PlayerManager) Start() {
	for _, player := range pm.players {
		player.restoreCurrentSnap()
	}
}

func (pm *PlayerManager) ApplyToAllPlayers(f func(player *Player)) {
	for _, player := range pm.players {
		f(player)
	}
}

func (pm *PlayerManager) ApplyToPlayersNamed(playerName string, f func(player *Player)) {
	for _, player := range pm.players {
		if playerName == player.playerName {
			f(player)
		}
	}
}

func (pm *PlayerManager) GetPlayer(playerName string) (*Player, error) {
	player, ok := pm.players[playerName]
	if !ok {
		return nil, fmt.Errorf("no player named %s", playerName)
	} else {
		return player, nil
	}
}

func (pm *PlayerManager) handleCursorEvent(ce CursorEvent) {
	pm.ApplyToAllPlayers(func(player *Player) {
		if player.IsSourceAllowed(ce.Source) {
			// call responders?
			player.HandleCursorEvent(ce)
		}
	})
}

func (pm *PlayerManager) handleMidiEvent(me MidiEvent) {
	pm.ApplyToAllPlayers(func(player *Player) {
		// if player.IsSourceAllowed(me.Source) {
		player.HandleMidiEvent(me)
		// }
	})
}
