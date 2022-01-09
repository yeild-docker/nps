package dynmicgateway

import (
	"strconv"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

const (
	DYNAMIC_CLIENT_ID = -1
	DYNAMIC_HOST_ID   = -1
	DYNAMIC_SOCKS_ID  = -1
)

var (
	Bridge_over_websocket string
	Dynamic_proxy_host    string
	Web_host              string
)

type DynmicGateway struct {
	DynamicHostSuffix string
}

var (
	DynG *DynmicGateway
)

func InitDynmicGateway() {
	DynG = &DynmicGateway{}
	Bridge_over_websocket = beego.AppConfig.String("bridge_over_websocket")
	Dynamic_proxy_host = beego.AppConfig.String("dynamic_proxy_host")
	Web_host = beego.AppConfig.String("web_host")
	DynG.DynamicHostSuffix = "." + Dynamic_proxy_host + "." + Web_host
}

func (dyn *DynmicGateway) ResolveHost(host string, scheme string) (dynamic bool, client_alias string, target_host string, target_port int, err error) {
	dynamic = false
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	if len(Dynamic_proxy_host) < 1 || !strings.HasSuffix(host, DynG.DynamicHostSuffix) {
		return
	}
	// {target-host}.{target-port}.{proxy client alias}
	t_host := strings.TrimSuffix(host, DynG.DynamicHostSuffix)
	p_hosts := strings.Split(t_host, ".")
	client_alias = p_hosts[len(p_hosts)-1]
	if len(p_hosts) == 2 {
		if target_port, err = strconv.Atoi(p_hosts[0]); err != nil {
			target_host = p_hosts[0]
		}
	} else if len(p_hosts) == 3 {
		if target_port, err = strconv.Atoi(p_hosts[1]); err != nil {
			return
		}
		target_host = p_hosts[0]
	}
	if len(target_host) > 0 && target_host != "local" {
		target_host = strings.Replace(target_host, "-", ".", -1)
	} else {
		target_host = "localhost"
	}
	if target_port == 0 {
		if strings.ToLower(scheme) == "http" {
			target_port = 80
		} else {
			target_port = 443
		}
	}
	logs.Info("target host: %s, port: %d, client: %s", target_host, target_port, client_alias)
	dynamic = true
	return
}
