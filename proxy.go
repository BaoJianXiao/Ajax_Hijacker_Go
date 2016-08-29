package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	Version = "0.1"
)

var (
	wg sync.WaitGroup
)

var logFile *os.File
var logger *log.Logger

func main() {
	var conf Cfg

	conf.Port = flag.String("port", "80", "Local Port")
	conf.Raddr = flag.String("raddr", "10.116.9.44:8089", "Remote IP & Port")
	conf.Static = flag.String("path", "D:\\works\\WeiXinTasks\\backend\\dist", "Static Resource Folder Path")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
	}

	wg.Add(1)
	html5proxy(&conf)
	wg.Wait()
}

func html5proxy(conf *Cfg) {
	handler := InitConfig(conf)
	server := &http.Server{
		Addr:         ":" + *conf.Port,
		Handler:      handler,
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
	}

	go func() {
		log.Println("Server Start For" + *conf.Raddr)
		server.ListenAndServe()
		wg.Done()
		log.Printf("Server Stop !")
	}()

	return
}
