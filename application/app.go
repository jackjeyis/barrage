package application

import (
	"errors"
	"flag"
	"os"
	"runtime/trace"
	"strconv"
	"sync"

	"barrage/logger"
	"barrage/msg"
	"barrage/network"
	"barrage/service"
	"barrage/util"
)

var (
	help_mode    = flag.Bool("h", false, "help mode")
	version_mode = flag.Bool("v", false, "version mode")
	daemon_mode  = flag.Bool("d", false, "daemon mode")
	app_config   = flag.String("c", ".", "app config")
	log_config   = flag.String("l", ".", "log config")
	ip_mode      = flag.Bool("i", false, "ip address")
)

type DelegateFunc func() error
type Application interface {
	OnStart() error
	OnStop() error
	OnFinish() error
}
type GenericApplication struct {
	io_service    *service.IOService
	logger_config string
	acceptors     []*network.Acceptor
	on_start      DelegateFunc
	on_stop       DelegateFunc
	on_finish     DelegateFunc
	app_config    string
	ip            string
}

func (app *GenericApplication) SetOnStart(fn DelegateFunc) *GenericApplication {
	app.on_start = fn
	return app
}

func (app *GenericApplication) SetOnStop(fn DelegateFunc) *GenericApplication {
	app.on_stop = fn
	return app
}

func (app *GenericApplication) SetOnFinish(fn DelegateFunc) *GenericApplication {
	app.on_finish = fn
	return app
}

func (app *GenericApplication) OnStart() error {
	if app.on_start != nil {
		return app.on_start()
	}
	return nil
}

func (app *GenericApplication) OnStop() error {
	if app.on_stop != nil {
		return app.on_stop()
	}
	return nil
}

func (app *GenericApplication) OnFinish() error {
	if app.on_finish != nil {
		return app.on_finish()
	}
	return nil
}

func (app *GenericApplication) RegisterService(name string, fn func(msg.Message)) {
	service.Instance().RegisterService(name, fn)
}

func (app *GenericApplication) Monitor() {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = trace.Start(f)
	if err != nil {
		panic(err)
	}
	defer trace.Stop()
}

func (app *GenericApplication) Run() (err error) {
	flag.Parse()
	if err = app.Start(); err != nil {
		logger.Info("Run error %v", err)
		return err
	}

	app.Monitor()
	/*if help_mode || version_mode {
		return nil
	}
	*/
	if err = app.Wait(); err != nil {
		return nil
	}

	app.Stop()

	return nil
}

func (app *GenericApplication) Start() (err error) {

	if err = app.ParseCmd(); err != nil {
		return nil
	}
	/*
		if help_mode || version_mode {
			return nil
		}
	*/
	if err = app.InitApp(); err != nil {
		return nil
	}

	if err = app.WritePidFile(); err != nil {
		return err
	}

	if err = app.StartLogger(); err != nil {
		return err
	}

	if err = app.LoadConfig(); err != nil {
		return err
	}

	if err = app.StartIOService(); err != nil {
		return err
	}

	if err = app.OnStart(); err != nil {
		return err
	}

	if err = app.StartServiceManager(); err != nil {
		return err
	}

	return nil
}

func (app *GenericApplication) Wait() (err error) {
	if err = app.RunIOService(); err != nil {
		return err
	}
	return nil
}

func (app *GenericApplication) Stop() (err error) {
	logger.Info("Application:OnStop")
	if err = app.OnStop(); err != nil {
		return err
	}

	logger.Info("Application:StopServiceManager")
	app.StopServiceManager()
	if err = app.StopIOService(); err != nil {
		return err
	}

	if err = app.OnFinish(); err != nil {
		return err
	}

	return nil
}

func (app *GenericApplication) ParseCmd() error {
	app.app_config = *app_config
	app.logger_config = *log_config
	if *ip_mode {
		app.ip = util.ExternalIp()
	} else {
		app.ip = util.InternalIp()
	}
	return nil
}

func (app *GenericApplication) InitApp() (err error) {
	if *daemon_mode {
		util.Daemon()
	}
	return nil
}

func (app *GenericApplication) Daemon() error {
	return nil
}

func (app *GenericApplication) WritePidFile() (err error) {
	var f *os.File
	defer f.Close()
	f, err = os.OpenFile("pid", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	f.WriteString(strconv.Itoa(os.Getpid()))
	return
}

func (app *GenericApplication) StartLogger() (err error) {
	var path string
	if app.logger_config == "" {
		path = "./log"
	} else {
		path = app.logger_config
	}
	if *daemon_mode {
		logger.Start(logger.LogFilePath(path), logger.EveryHour, logger.PrintStack)
	} else {
		logger.Start(logger.EveryHour, logger.PrintStack)
	}
	return nil
}

func (app *GenericApplication) LoadConfig() (err error) {
	if app.app_config != "" {
		util.NewConfigManager(app.app_config)
		logger.Info("app conf path %s", util.GetConf())
	} else {

	}
	return nil
}

func (app *GenericApplication) StartIOService() (err error) {
	var io_service_config service.IOServiceConfig
	app.io_service = app.GetIOService()
	cf := util.GetIOServiceConf()
	if cf.Srvworker != 0 {
		io_service_config.Srvworker = cf.Srvworker
		io_service_config.Srvqueue = cf.Srvqueue
		io_service_config.Ioworker = cf.Ioworker
		io_service_config.Ioqueue = cf.Ioqueue
		io_service_config.Matrixbucket = cf.Matrixbucket
		io_service_config.Matrixsize = cf.Matrixsize
	} else {
		io_service_config.Srvworker = 1000
		io_service_config.Srvqueue = 10000
		io_service_config.Ioworker = 1000
		io_service_config.Ioqueue = 10000
		io_service_config.Matrixbucket = 16
		io_service_config.Matrixsize = 1024
	}
	if err = app.io_service.Init(io_service_config); err != nil {
		return err
	}

	app.io_service.HandleSignal()
	app.io_service.Start()
	return nil
}

var (
	io_srv *service.IOService
	once   sync.Once
)

func (app *GenericApplication) GetIOService() *service.IOService {
	once.Do(func() {
		io_srv = &service.IOService{}
	})
	return io_srv
}

func (app *GenericApplication) StartServiceManager() (err error) {
	if util.GetManager() == nil {
		return errors.New("Application.StartServiceManager no config set,ignore")
	}

	var acceptor *network.Acceptor
	logger.Info("service config %v", util.GetServicesConfig())
	for _, service_config := range util.GetServicesConfig() {
		acceptor, err = network.Instance().CreateAcceptor(app.GetIOService(),
			service_config.Addr, service_config.Name)

		if err != nil {
			logger.Error("Application.StartServiceManager IODescriptorFactory.CreateAcceptor fail,address:%s,name:%s",
				service_config.Addr, service_config.Name)
			return err
		}

		go acceptor.Start()

		app.acceptors = append(app.acceptors, acceptor)
	}

	return nil
}

func (app *GenericApplication) RunIOService() error {
	app.io_service.Run()
	return nil
}

func (app *GenericApplication) StopServiceManager() error {
	for _, acceptor := range app.acceptors {
		acceptor.Stop()
	}
	return nil
}

func (app *GenericApplication) StopIOService() error {
	app.io_service.CleanUp()
	return nil
}

func (app *GenericApplication) GetIp() string {
	return app.ip
}
