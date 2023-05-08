package utils

import (
	"k8s.io/klog/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"time"
)

type ProxyHolder struct {
	proxy *httputil.ReverseProxy
	host  string
}

type ProxyPool struct {
	Port       string
	HostTarget map[string]string
	HostProxy  map[string]*ProxyHolder
	IpfsHosts  []string
}

func (h *ProxyPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	host := r.Host

	klog.Info("Request", host)
	klog.Info("Mapping", h.HostTarget)
	klog.Info("Proxies", h.HostProxy)

	// Define the regular expression to match URLs that need to be rewritten
	re := regexp.MustCompile("^(.*/)$")

	// Check if the incoming request matches the regular expression
	if re.MatchString(r.URL.Path) {
		// Rewrite the URL to remove the "/api" prefix
		r.URL.Path = re.ReplaceAllString(r.URL.Path, "/")
	}

	if fn, ok := h.HostProxy[host]; ok {
		klog.Infof("Serve: %", fn.host)
		r.Host = fn.host
		fn.proxy.ServeHTTP(w, r)
		return
	}

	if target, ok := h.HostTarget[host]; ok {
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
		h.HostProxy[host] = &ProxyHolder{proxy: proxy, host: r.Host}

		return
	}
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("403: Host forbidden " + host))
}

func (h *ProxyPool) Start() {

	mux := http.NewServeMux()

	klog.Info("Start proxy server on port:", h.Port)

	conf := map[string][]string{
		"menu":    []string{"children"},
		"article": []string{"items", "content"}}

	// todo get from config
	dataCache := NewDagCache(h.IpfsHosts, 10*time.Hour, 20, conf)

	mux.Handle("/", h)

	mux.HandleFunc("/dag", func(writer http.ResponseWriter, request *http.Request) {

		key := request.URL.Query().Get("key")
		cid := request.URL.Query().Get("cid")
		resp0, err := dataCache.ProcessQuery(key, cid)
		if err != nil {
			panic(err)
		}

		writer.Write(resp0)
	})

	klog.Fatal(http.ListenAndServe(":"+h.Port, mux))
}
