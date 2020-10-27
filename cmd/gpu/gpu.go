package main

import (
	"flag"
	"fmt"
	"github.com/spacemeshos/post/gpu"
)

var (
	id      = make([]byte, 32)
	salt    = make([]byte, 32)
	options = uint32(0)
)

var provider = flag.Int("provider", 0, "provider")
var hlen = flag.Int("hlen", 0, "hlen")
var positions = flag.Int("positions", 0, "positions")

func main() {
	//providers := gpu.Providers()
	//for _, p := range providers {
	//	providerId := uint(p.ID)
	//	startPosition := uint64(1)
	//	endPosition := uint64(1 << 8)
	//	hashLenBits := uint32(4)
	//	output, _, _ := gpu.ScryptPositions(providerId, id, salt, startPosition, endPosition, hashLenBits, options)
	//
	//	fmt.Printf("provider: %+v, output: %x\n", p, output)
	//}

	//println()

	flag.Parse()
	startPosition := uint64(1)
	endPosition := uint64(*positions)
	output, _, _ := gpu.ScryptPositions(uint(*provider), id, salt, startPosition, endPosition, uint32(*hlen), options)

	fmt.Printf("provider: %v, output: %x\n", *provider, output[:8])

}
