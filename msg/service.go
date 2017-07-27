package msg

type Service interface {
	Serve(Message)
}

type Stage interface {
        Start()
        Wait()
        Stop()
        Send(Message)
}
