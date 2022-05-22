//cinfo 1 acd63266fdfc8a19f5f69f8ca68a2f9a
package main

import (
	log "log"
	engine "github.com/vizicist/palette/engine"
)

func callback(eventType string, eventData string) {
//line C:\Users\tjt\Documents\GitHub\palette\plugins\example_gop\example_gop.gop:10
	log.Printf("hello: callback type=%s data=%s\n", eventType, eventData)
}
func main() {
//line C:\Users\tjt\Documents\GitHub\palette\plugins\example_gop\example_gop.gop:14
	err := engine.Register("hello", engine.EventAll, callback)
//line C:\Users\tjt\Documents\GitHub\palette\plugins\example_gop\example_gop.gop:15
	if err != nil {
//line C:\Users\tjt\Documents\GitHub\palette\plugins\example_gop\example_gop.gop:16
		log.Printf("plugin.Register: err=%s\n", err)
//line C:\Users\tjt\Documents\GitHub\palette\plugins\example_gop\example_gop.gop:17
		return
	}
//line C:\Users\tjt\Documents\GitHub\palette\plugins\example_gop\example_gop.gop:19
	select {}
}
