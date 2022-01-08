package proxy

import (
	"net/http"
	"strconv"
	"strings"

	"ehang.io/nps/lib/file"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

const (
	DYNAMIC_CLIENT_ID = -1
	DYNAMIC_HOST_ID   = -1
	DYNAMIC_SOCKS_ID  = -1
)

type DynmicGateway struct {
	Bridge_over_websocket string
	Dynamic_proxy_host    string
	Web_host              string
}

var (
	S_DynmicGateway *DynmicGateway
)

func GenDynmicGatewayClient() *file.Client {
	return &file.Client{
		Id:     DYNAMIC_HOST_ID,
		Status: false,
		Remark: "Dynamic",
		Alias:  "local",
		Flow:   &file.Flow{},
		Cnf: &file.Config{
			Compress: true,
			Crypt:    true,
		},
	}
}

func GenDynmicGatewayHost() *file.Host {
	return &file.Host{
		Id:           DYNAMIC_CLIENT_ID,
		Host:         "*." + S_DynmicGateway.Dynamic_proxy_host + "." + S_DynmicGateway.Web_host,
		Target:       &file.Target{TargetStr: "", LocalProxy: true},
		HeaderChange: "",
		HostChange:   "",
		Remark:       "",
		Location:     "/",
		Flow:         &file.Flow{},
		Scheme:       "all",
		KeyFilePath:  "",
		CertFilePath: "",
	}
}

func InitDynmicGateway() {
	S_DynmicGateway = &DynmicGateway{}
	S_DynmicGateway.Bridge_over_websocket = beego.AppConfig.String("bridge_over_websocket")
	S_DynmicGateway.Dynamic_proxy_host = beego.AppConfig.String("dynamic_proxy_host")
	S_DynmicGateway.Web_host = beego.AppConfig.String("web_host")
	var err error
	if len(S_DynmicGateway.Dynamic_proxy_host) > 0 {
		var host *file.Host
		if host, err = file.GetDb().GetHostById(DYNAMIC_CLIENT_ID); err != nil {
			host = GenDynmicGatewayHost()
		}
		isNewHost := err != nil
		if host.Client, err = file.GetDb().GetClient(DYNAMIC_HOST_ID); err != nil {
			host.Client = GenDynmicGatewayClient()
			err = file.GetDb().NewClient(host.Client)
			if err != nil {
				logs.Error("add dynamic client fail: " + err.Error())
			}
		}
		if isNewHost {
			err = file.GetDb().NewHost(host)
			if err != nil {
				logs.Error("add dynamic host fail: " + err.Error())
			}
		}
	}
}

func (dyn *DynmicGateway) HandleHost(host *file.Host, r *http.Request) (bool, error) {
	var err error
	dyn_suffix := "." + dyn.Dynamic_proxy_host + "." + beego.AppConfig.String("web_host")
	request_host := strings.Split(r.Host, ":")[0]
	var client_alias string
	var target_host string
	target_port := 0
	changed := false
	if host.Client.Id == -1 && len(dyn.Dynamic_proxy_host) > 0 && strings.HasSuffix(request_host, dyn_suffix) {
		// {target-host}.{target-port}.{proxy client alias}
		t_host := strings.TrimSuffix(request_host, dyn_suffix)
		p_hosts := strings.Split(t_host, ".")
		client_alias = p_hosts[len(p_hosts)-1]
		if len(p_hosts) == 2 {
			if target_port, err = strconv.Atoi(p_hosts[0]); err != nil {
				target_host = p_hosts[0]
			}
		} else if len(p_hosts) == 3 {
			if target_port, err = strconv.Atoi(p_hosts[1]); err != nil {
				logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
				return false, err
			}
			target_host = p_hosts[0]
		}
		if len(target_host) > 0 {
			target_host = strings.Replace(target_host, "-", ".", -1)
		} else {
			target_host = "localhost"
		}
		if target_port == 0 {
			if strings.ToLower(r.URL.Scheme) == "http" {
				target_port = 80
			} else {
				target_port = 443
			}
		}
		logs.Info("target host: %s, port: %d, client: %s", target_host, target_port, client_alias)
		host.HostChange = target_host
		host.Target.TargetStr = target_host + ":" + strconv.Itoa(target_port)
		host.Target.LocalProxy = client_alias == "local"
		changed = true
		if host.Client, err = file.GetDb().JsonDb.GetClientByAlias(client_alias); err != nil {
			if !host.Target.LocalProxy {
				logs.Notice("the client %s not found, error %s", client_alias, err)
				return false, err
			}
			host.Client = GenDynmicGatewayClient()
		} else {
			host.Client.Cnf.Crypt = !host.Target.LocalProxy
			host.Client.Cnf.Compress = !host.Target.LocalProxy
		}
	}
	return changed, nil
}
