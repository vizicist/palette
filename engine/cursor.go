package engine

import (
	"log"
	"sync"
	"time"
)

type CursorManager struct {
	cursors map[string]*DeviceCursor
	mutex   sync.RWMutex
}

func NewCursorManager() *CursorManager {
	return &CursorManager{
		cursors: map[string]*DeviceCursor{},
		mutex:   sync.RWMutex{},
		// responders: make(map[string]CursorResponder),
	}
}

func (cm *CursorManager) clearCursors() {

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for id, c := range cm.cursors {
		if !c.downed {
			log.Printf("Hmmm, why is a cursor not downed?\n")
		} else {
			cm.handleCursorEventNoLock(CursorEvent{Source: id, Ddu: "up"})
			if Debug.Cursor {
				log.Printf("Clearing cursor id=%s\n", id)
			}
			delete(cm.cursors, id)
		}
	}
}

func (cm *CursorManager) handleCursorEvent(ce CursorEvent) {
	cm.mutex.Lock()
	defer func() {
		cm.mutex.Unlock()
	}()
	cm.handleCursorEventNoLock(ce)
}

func (cm *CursorManager) handleCursorEventNoLock(ce CursorEvent) {

	// As soon as there's any non-internal cursor event,
	// we turn attract mode off.
	if ce.Source != "internal" {
		go func() {
			TheEngine().Scheduler.Control <- Command{"attractmode", false}
		}()
		// TheEngine().Scheduler.SetAttractMode(false)
	}

	switch ce.Ddu {

	case "clear":
		// Special event to clear cursors (by sending them "up" events)
		// XXX - should this go in cursor.go?  yes.
		cm.clearCursors()
		return

	case "down", "drag", "up":

		c, ok := cm.cursors[ce.ID]
		if !ok {
			// new DeviceCursor
			c = &DeviceCursor{}
			cm.cursors[ce.ID] = c
		}
		c.lastTouch = time.Now()

		// If it's a new (downed==false) cursor, make sure the first event is "down"
		if !c.downed {
			ce.Ddu = "down"
			c.downed = true
		}
		if Debug.Cursor {
			log.Printf("CursorManager.handleCursorEvent: id=%s ddu=%s xyz=%.4f,%.4f,%.4f\n", ce.ID, ce.Ddu, ce.X, ce.Y, ce.Z)
		}

		if ce.Ddu == "up" {
			delete(cm.cursors, ce.ID)
		}
	}
}

func (cm *CursorManager) doCursorGesture(source string, cid string, x0, y0, z0, x1, y1, z1 float32) {
	log.Printf("doCursorGesture: start\n")

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	ce := CursorEvent{
		Source:    source,
		Timestamp: time.Now(),
		Ddu:       "down",
		X:         x0,
		Y:         y0,
		Z:         z0,
		Area:      0,
	}
	cm.handleCursorEventNoLock(ce)
	if Debug.Cursor {
		log.Printf("doCursorGesture: down ce=%+v\n", ce)
	}

	// secs := float32(3.0)
	secs := float32(TheEngine().Scheduler.attractNoteDuration)
	dt := time.Duration(int(secs * float32(time.Second)))
	time.Sleep(dt)
	ce.Ddu = "up"
	ce.X = x1
	ce.Y = y1
	ce.Z = z1
	cm.handleCursorEventNoLock(ce)
	if Debug.Cursor {
		log.Printf("doCursorGesture: up ce=%+v\n", ce)
	}
	log.Printf("doCursorGesture: end\n")
}

func (cm *CursorManager) checkCursorUp(now time.Time) {

	if checkDelay == 0 {
		milli := ConfigIntWithDefault("upcheckmillisecs", 1000)
		checkDelay = time.Duration(milli) * time.Millisecond
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for id, c := range cm.cursors {
		elapsed := now.Sub(c.lastTouch)
		if elapsed > checkDelay {
			cm.handleCursorEventNoLock(CursorEvent{Source: "checkCursorUp", Ddu: "up"})
			if Debug.Cursor {
				log.Printf("Player.checkCursorUp: deleting cursor id=%s elapsed=%v\n", id, elapsed)
			}
			delete(cm.cursors, id)
		}
	}
}
