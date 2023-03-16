package main

import (
	"k8s.io/klog/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type ProxyStruct struct {
	proxy *httputil.ReverseProxy
	host  string
}

type ProxyHandlers struct {
	hostTarget map[string]string
	hostProxy  map[string]*ProxyStruct
}

func (h *ProxyHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host

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
		h.hostProxy[host] = &ProxyStruct{proxy: proxy, host: r.Host}

		return
	}
	w.Write([]byte("403: Host forbidden " + host))
}

func updateConfig(map[string]string) {
	// TODO: Update the configuration data in your application
}

func main() {

	h := &ProxyHandlers{
		hostTarget: map[string]string{
			"uimatrix.solenopsys.org": "http://ipfs.alpha.solenopsys.org/ipfs/QmaLUcpQVs5QdVAHCB6D2C524tMEFors9WkcLRm5BAfh4T/",
		},
		hostProxy: map[string]*ProxyStruct{},
	}

	http.Handle("/", h)

	server := &http.Server{
		Addr:    ":80",
		Handler: h,
	}
	klog.Fatal(server.ListenAndServe())
}
