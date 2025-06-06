package utilities

import (
	"log"
	"os"
)

var Logger *log.Logger

func init() {
	Logger = log.New(os.Stdout, "[Passenger] ", log.LstdFlags|log.Lshortfile)
}
