package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/lab46/example/bookapp/book"
	"github.com/lab46/example/pkg/env"
	"github.com/lab46/example/pkg/flags"
	"github.com/lab46/example/pkg/log"
	"github.com/lab46/example/pkg/rdbms"
	"github.com/lab46/example/rentapp/service"
)

func initService() (*service.Service, error) {
	serviceConfig, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	masterDB, err := rdbms.Open("postgres", serviceConfig.Postgres.MasterExampleDB)
	if err != nil {
		return nil, err
	}
	slaveDB, err := rdbms.Open("postgres", serviceConfig.Postgres.SlaveExampleDB)
	if err != nil {
		return nil, err
	}

	// init package
	book.Init(masterDB, rdbms.NewLoadBalancer(slaveDB))
	// create new service
	s := service.New("9000")
	return s, err
}

type serviceFlags struct {
	logLevel  string
	configDir string
}

var sf serviceFlags

func (sf *serviceFlags) Parse(fs *flag.FlagSet, args []string) error {
	fs.StringVar(&sf.logLevel, "log_level", "", "logging level")
	fs.StringVar(&sf.configDir, "config_dir", "", "configuration directory")
	return fs.Parse(args)
}

func main() {
	sf = serviceFlags{}
	flags.Parse(&sf)
	log.SetLevelString(sf.logLevel)
	env.SetConfigDir(sf.configDir)

	s, err := initService()
	if err != nil {
		log.Fatal(err)
	}

	fatalChan := make(chan error)
	go func() {
		fatalChan <- s.RunWebserver()
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		log.Println("Signal terminate detected")
	case err := <-fatalChan:
		log.Fatal("Application failed to run because ", err.Error())
	}
}
