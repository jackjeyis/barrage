package main

import (
	"barrage/application"
	"barrage/logger"
	"barrage/msg"
	"barrage/network"
	"barrage/protocol"
	"barrage/util"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
	//"github.com/samuel/go-zookeeper/zk"
	//"github.com/stackimpact/stackimpact-go"
)

type Auth struct {
	Rid   string
	Token string
	Cid   string
}

type AuthReply struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Gag  bool   `json:"gag"`
}

var (
	app *application.GenericApplication
)

var m sync.Mutex

func main() {

	/*agent := stackimpact.NewAgent()
	agent.Start(stackimpact.Options{
		AgentKey:       "67e7727fd85ce05c93889fccbc6f6444e045eba3",
		AppName:        "Basic Go Server",
		AppVersion:     "1.0.0",
		AppEnvironment: "production",
	})
	*/

	go func(){
		logger.Info("http listen success!")
		http.ListenAndServe(":7777",nil)
	}()
	app = &application.GenericApplication{}
	app.SetOnStart(func() error {
		app.RegisterService("mqtt_handler", func(msg msg.Message) {
			switch msg := msg.(type) {
			case *protocol.MqttConnect:
				logger.Info("connect success!")
				//network.Register(msg.ClientId, msg.Channel())
				m := &protocol.MqttConnAck{RetCode: protocol.RetCodeAccepted}
				//m.SetChannel(network.GetSession(msg.ClientId))
				m.SetChannel(msg.Channel())
				app.GetIOService().GetIOStage().Send(m)
			case *protocol.MqttPublish:
				//logger.Info("publish %v", msg)
				//m := &protocol.MqttPubAck{MsgId: msg.MsgId}
				//m.SetChannel(msg.Channel())
				//app.GetIOService().GetIOStage().Send(m)
				/*submit := &pb.Submit{}
				err := proto.Unmarshal(msg.Topic, submit)
				if err != nil {
					logger.Error("Unmarshal error %v", err)
				}
				logger.Info("Topic To %v", submit.To)
				*/
				app.GetIOService().GetIOStage().Send(msg)

			case *protocol.MqttPingReq:
				logger.Info("ping req")
				m := &protocol.MqttPingRes{}
				m.SetChannel(msg.Channel())
				app.GetIOService().GetIOStage().Send(m)

			}
		})
		app.RegisterService("barrage_handler", func(msg msg.Message) {
			barrage := msg.(*protocol.Barrage)
			if barrage == nil {
				return
			}

			if barrage.Channel().GetAttr("status") != "OK" {
				if barrage.Op != 7 {
					logger.Info("handshake fail!")
					barrage.Channel().Close()
					return
				}
				logger.Info("auth %v", string(barrage.Body))
				var auth Auth
				if err := json.Unmarshal(barrage.Body, &auth); err != nil {
					logger.Error("json.Unmarshal error %v,body %v", err, string(barrage.Body))
					barrage.Channel().Close()
					return
				}
				if auth.Rid == "" || auth.Cid == "" || auth.Token == "" {
					logger.Error("param invalid")
					barrage.Channel().Close()
					return
				}
				//res, err := util.Get("http://" + util.GetHttpConfig().Remoteaddr + "/im/" + auth.Rid + "/_check?token=" + auth.Token)
				var err error
				if err != nil {
					logger.Error("check token api error %v", err)
					barrage.Channel().Close()
					return
				}
				//logger.Info("res %v", res)

				var reply AuthReply

				switch 0 {
				case 0:
					reply.Code = 0
					reply.Msg = "鉴权成功!"
					reply.Gag = true
				case 403:
					reply.Code = 1
					reply.Msg = "token 校验失败!"
				case 1001:
					reply.Code = 2
					reply.Msg = "该直播已被管理员删除!"
				case 1002:
					reply.Code = 3
					reply.Msg = "您已被管理员移出，无权限观看!"
				}
				ch := network.GetSession(auth.Cid)
				if ch != nil {
					reply.Code = 0
					reply.Msg = "该用户重新连接,踢出老的登录设备!"
					ch.SetAttr("status", "")
					ch.Close()
					network.UnRegister(auth.Cid, auth.Cid, auth.Rid)
				}

				body, _ := util.EncodeJson(reply)
				b := &protocol.Barrage{}
				b.Op = 8
				b.Ver = 1
				b.Body = body
				if reply.Code != 0 {
					barrage.Channel().Send(barrage)
					time.Sleep(1 * time.Second)
					barrage.Channel().Close()
					return
				}
				barrage.Channel().SetAttr("status", "OK")
				barrage.Channel().SetAttr("cid", auth.Cid)
				barrage.Channel().SetAttr("uid", auth.Cid)
				barrage.Channel().SetAttr("rid", auth.Rid)
				barrage.Channel().SetAttr("ct", util.GetClientType(auth.Cid))
				network.Register(auth.Cid, barrage.Channel(), 1)
				barrage.Channel().Send(b)
				network.NotifyHost(auth.Rid, auth.Cid, auth.Cid, 1)
			}
			switch barrage.Op {
			case 2:
				barrage.Op = 3
				barrage.Channel().SetDeadline(240)
				barrage.Channel().Send(barrage)
			case 4:
				barrage.Op = 5
				network.BroadcastRoom(barrage, true)
			}
		})

		go func() {
			httpServeMux := http.NewServeMux()
			httpServeMux.HandleFunc("/1/push/room", PushRoom)
			httpServeMux.HandleFunc("/1/get/room", GetRoom)
			httpServer := &http.Server{
				Handler:      httpServeMux,
				ReadTimeout:  time.Duration(5) * time.Second,
				WriteTimeout: time.Duration(5) * time.Second,
			}
			httpServer.SetKeepAlivesEnabled(true)
			addr := util.GetHttpConfig().Localaddr
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				logger.Error("net.Listen %s error %v", addr)
				panic(err)
			}
			logger.Info("start http listen %s", addr)
			if err = httpServer.Serve(ln); err != nil {
				logger.Error("httpServer.Serve() error %v", err)
				panic(err)
			}
		}()

		/*go func() {
			c, ev, err := zk.Connect(util.GetZkConfig().Addrs, util.GetZkConfig().Timeout.Duration)
			if err != nil {
				logger.Error("zk.Connect (\"%v\") error (%v)", util.GetZkConfig().Addrs, err)
				panic(err)
			}
			go func(ev <-chan zk.Event) {
				for e := range ev {
					if e.State == zk.StateHasSession {
						var path string
						path = "/barrage"
						path, err = c.Create(path, []byte(""), 0, zk.WorldACL(zk.PermAll))
						//		path = "/barrage/server"
						//		path, err = c.Create(path, []byte(""), 0, zk.WorldACL(zk.PermAll))
						paths, _, err := c.Children("/barrage")
						if len(paths) == 0 {
							path, err = c.Create("/barrage/master", []byte(app.GetIp()), zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
							if err != nil {
								logger.Error("zk.Create (\"%s\") error (%v)", path, err)
								//panic(err)
							}
						}
						go func() {
							for {
								paths, _, ev, err := c.ChildrenW("/barrage")
								e := <-ev
								if e.Type == zk.EventNodeChildrenChanged {

									if len(paths) == 1 {
										logger.Info("event %s type %v paths %v", e.State.String(), e.Type, paths)
										path, err = c.Create("/barrage/master", []byte(app.GetIp()), zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
										if err != nil {
											logger.Error("zk.Create Watch (\"%s\"), error (%v)", path, err)
										}
									}
								}
							}
						}()

					}
				}
			}(ev)
		}()
		*/
		return nil
	}).Run()

}

func retWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		logger.Error("json.Marshal error %v", err)
		return
	}
	if _, err := w.Write(data); err != nil {
		logger.Error("w.Write error %v", err)
	}
	//logger.Info("req: \"%s\",get: ip:\"%s\",time:\"%fs\"", r.URL.String(), r.RemoteAddr, time.Now().Sub(start).Seconds())
}

func PushRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	var (
		bodyBytes []byte
		err       error
		param     = r.URL.Query()
		res       = map[string]interface{}{"ret": 1}
	)

	defer retWrite(w, r, res, time.Now())
	w.Header().Set("Content-Type", "application/json")
	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		logger.Error("ioutil.ReadAll failed %v", err)
		res["ret"] = 65535
		return
	}
	barrage := &protocol.Barrage{}
	barrage.Op = 5
	barrage.Ver = 1
	barrage.Body = bodyBytes
	c := network.NewChannel()
	c.SetAttr("rid", param.Get("rid"))
	barrage.SetChannel(c)
	network.BroadcastRoom(barrage, false)
	return
}

func GetRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	var (
		param = r.URL.Query()
		res   = map[string]interface{}{"ret": 1}
	)

	defer retWrite(w, r, res, time.Now())
	w.Header().Set("Content-Type", "application/json")
	count, uids := network.GetRoomStatus(param.Get("rid"))
	if uids == nil {
		res["ret"] = 2
		return
	}
	res["data"] = uids
	res["total"] = count
	return
}
