package main

import (
	"log"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
}

func main() {
	cli := CLI{}
	cli.Run()
}
