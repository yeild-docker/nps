package file

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/server/dynmicgateway"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

var (
	CachedHosts sync.Map
)

func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return json.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func GenDynmicGatewayClient() *Client {
	return &Client{
		Id:        dynmicgateway.DYNAMIC_HOST_ID,
		Status:    true,
		IsConnect: true,
		Remark:    "Dynamic",
		Alias:     "local",
		Flow:      &Flow{},
		Cnf: &Config{
			Compress: false,
			Crypt:    false,
		},
	}
}

func GenDynmicGatewayHost() *Host {
	return &Host{
		Id:           dynmicgateway.DYNAMIC_CLIENT_ID,
		Host:         "*." + dynmicgateway.Dynamic_proxy_host + "." + dynmicgateway.Web_host,
		Target:       &Target{TargetStr: "", LocalProxy: true},
		HeaderChange: "",
		HostChange:   "",
		Remark:       "",
		Location:     "/",
		Flow:         &Flow{},
		Scheme:       "all",
		KeyFilePath:  "",
		CertFilePath: "",
	}
}

func InitDynmicGateway() {
	if opened, _ := beego.AppConfig.Bool("web_open_ssl"); opened {
		web_key_file := beego.AppConfig.String("web_key_file")
		if len(web_key_file) > 0 && !strings.HasPrefix(web_key_file, "/") {
			beego.AppConfig.Set("web_key_file", filepath.Join(common.GetRunPath(), web_key_file))
		}
		web_cert_file := beego.AppConfig.String("web_cert_file")
		if len(web_cert_file) > 0 && !strings.HasPrefix(web_cert_file, "/") {
			beego.AppConfig.Set("web_cert_file", filepath.Join(common.GetRunPath(), web_cert_file))
		}
	}
	https_default_cert_file := beego.AppConfig.String("https_default_cert_file")
	if len(https_default_cert_file) > 0 && !strings.HasPrefix(https_default_cert_file, "/") {
		beego.AppConfig.Set("https_default_cert_file", filepath.Join(common.GetRunPath(), https_default_cert_file))
	}
	https_default_key_file := beego.AppConfig.String("https_default_key_file")
	if len(https_default_key_file) > 0 && !strings.HasPrefix(https_default_key_file, "/") {
		beego.AppConfig.Set("https_default_key_file", filepath.Join(common.GetRunPath(), https_default_key_file))
	}
	dynmicgateway.InitDynmicGateway()
	var err error
	if len(dynmicgateway.Dynamic_proxy_host) > 0 {
		var host *Host
		if host, err = GetDb().GetHostById(dynmicgateway.DYNAMIC_CLIENT_ID); err != nil {
			host = GenDynmicGatewayHost()
		}
		isNewHost := err != nil
		if host.Client, err = GetDb().GetClient(dynmicgateway.DYNAMIC_HOST_ID); err != nil {
			host.Client = GenDynmicGatewayClient()
			err = GetDb().NewClient(host.Client)
			if err != nil {
				logs.Error("add dynamic client fail: " + err.Error())
			}
		}
		if isNewHost {
			err = GetDb().NewHost(host)
			if err != nil {
				logs.Error("add dynamic host fail: " + err.Error())
			}
		}
	}
}

func HandleDynamicHost(host *Host, r *http.Request) (*Host, error) {
	if host != nil && host.Id != dynmicgateway.DYNAMIC_HOST_ID {
		return host, nil
	}
	request_host := strings.Split(r.Host, ":")[0]
	if v, ok := CachedHosts.Load(r.Host); ok {
		return v.(*Host), nil
	}
	dynamic, client_alias, target_host, target_port, err := dynmicgateway.DynG.ResolveHost(request_host, r.URL.Scheme)
	if err != nil {
		logs.Warn("ResolveHost failed: %s", err)
		return nil, err
	}
	if !dynamic {
		return host, nil
	}
	// var h *Host
	h := &Host{}
	isNewHost := false
	if host != nil {
		DeepCopy(h, host)
	} else {
		if h, err = GetDb().GetHostById(dynmicgateway.DYNAMIC_HOST_ID); err != nil {
			isNewHost = true
			h = GenDynmicGatewayHost()
		}
	}
	h.HostChange = target_host
	h.Target.TargetStr = target_host + ":" + strconv.Itoa(target_port)
	h.Target.LocalProxy = client_alias == "local"
	if h.Client, err = GetDb().JsonDb.GetClientByAlias(client_alias); err != nil {
		if !host.Target.LocalProxy {
			logs.Notice("the client %s not found, error %s", client_alias, err)
			return nil, err
		}
		h.Client = GenDynmicGatewayClient()
	} else {
		DeepCopy(h.Client, h.Client)
		h.Client.Cnf.U = ""
		h.Client.Cnf.P = ""
		h.Client.Cnf.Compress = !h.Target.LocalProxy
	}
	if isNewHost {
		GetDb().NewHost(h)
	}
	CachedHosts.Store(h.Host, h)
	return h, nil
}
