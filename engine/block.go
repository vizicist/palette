package engine

// Block xxx
type Block interface {
	AcceptEngineMsg(ctx *EContext, cmd Cmd) string
	// Do(cmd string, arg interface{}) (interface{}, error)
}

// EContext xxx
type EContext struct {
	// instanceName string
	// blockType    string
	wantsClick bool
}

// Wire xxx
// type Wire struct {
// 	fromBlock string
// 	fromPort  string
// 	toBlock   string
// 	toPort    string
// }

// BlockMakerFunc xxx
type BlockMakerFunc func(ctx *EContext) Block

// BlockMaker xxx
var BlockMaker = map[string]BlockMakerFunc{}

// RegisterBlock xxx
func RegisterBlock(blockName string, f BlockMakerFunc) {
	BlockMaker[blockName] = f
}
