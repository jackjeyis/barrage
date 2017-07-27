package ctx

var context Context

type Context struct {
	isrv   *IOService
	hander *ServiceHandler
}

func Init(isrv *IOService, h *ServiceHandler) {
	context = Context{}
	context.isrv = isrv
	context.hander = h
}
