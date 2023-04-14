package main

import (
	"k8s.io/klog/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type ProxyHolder struct {
	proxy *httputil.ReverseProxy
	host  string
}

type ProxyPool struct {
	port       string
	hostTarget map[string]string
	hostProxy  map[string]*ProxyHolder
}

func (h *ProxyPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	host := r.Host

	klog.Info("Request", host)
	klog.Info("Mapping", h.hostTarget)
	klog.Info("Proxies", h.hostProxy)

	if fn, ok := h.hostProxy[host]; ok {
		klog.Infof("Serve: %", fn.host)
		r.Host = fn.host
		fn.proxy.ServeHTTP(w, r)
		return
	}

	if target, ok := h.hostTarget[host]; ok {
		remoteUrl, err := url.Parse(target)
		klog.Infof("process url: %", remoteUrl.Path)
		if err != nil {
			klog.Errorf("target parse fail:", err)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		r.Host = remoteUrl.Host
		klog.Errorf("host:", r.Host)
		proxy.ServeHTTP(w, r)
		h.hostProxy[host] = &ProxyHolder{proxy: proxy, host: r.Host}

		return
	}
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("403: Host forbidden " + host))
}

func (h *ProxyPool) Start() {
	http.Handle("/", h)

	server := &http.Server{
		Addr:    ":80",
		Handler: h,
	}
	klog.Fatal(server.ListenAndServe())
}
