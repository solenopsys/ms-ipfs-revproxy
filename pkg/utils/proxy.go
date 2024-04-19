package utils

import (
	"net/http"
	"net/http/httputil"

	"net/url"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"k8s.io/klog/v2"
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

	klog.Info("Request ", host)
	klog.Info("Mapping ", h.HostTarget)
	//klog.Info("Proxies", h.HostProxy)

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
		// proxy.Director = func(req *http.Request) {
		// 	req.URL.Path = "/index.html"
		// }
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

	r := mux.NewRouter()

	klog.Info("Start proxy server on port:", h.Port)

	conf := map[string][]string{
		"group":   {"article", "children", "value"},
		"menu":    {"children"},
		"article": {"items", "content"},
		"md":      {"children", "value"},
	}

	// todo get from config
	httpLoader := NewHttpLoader(h.IpfsHosts, 10)
	dataCache := NewDagCache(httpLoader, 10*time.Hour, conf)
	sharedCache := NewSharedCache(httpLoader, 10*time.Hour)

	t := 10 * time.Second
	plainCache := cache.New(t, t*2)
	go sharedCache.LoadMappingAll()
	valueSelector := "microfrontend"
	modulesMapping := NewPinningMapping("type", valueSelector, "name", true)
	go modulesMapping.LoadMapping(valueSelector)

	r.HandleFunc("/dag", func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("key")
		cid := request.URL.Query().Get("cid")
		resp0, err := dataCache.ProcessQuery(key, cid)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Write(resp0)
	}).Methods("GET")

	r.HandleFunc("/cached", func(writer http.ResponseWriter, request *http.Request) {
		cid := request.URL.Query().Get("cid")

		if data, found := plainCache.Get(cid); found {

			klog.Info("From cache: ", cid)
			writer.Write(data.([]byte))
		} else {
			bytes, err := httpLoader.httpGetJson(cid, false)

			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			plainCache.Set(cid, bytes, cache.DefaultExpiration)

			writer.Write(bytes)
		}

	}).Methods("GET")

	r.HandleFunc("/shared/{libName}", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		libName := vars["libName"]
		resp0, err := sharedCache.GetLib(libName)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/javascript")
		writer.Write(resp0)

	}).Methods("GET")

	r.HandleFunc("/modules/{company}/{module}/{file}", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		company := vars["company"]
		module := vars["module"]
		file := vars["file"]

		cid, err := modulesMapping.GetCid("@" + company + "/" + module)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusNotFound)
			return
		}
		body, err := httpLoader.httpGetSubFile(cid, "/"+file, true, false)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/javascript")
		writer.Write(body)

	}).Methods("GET")

	r.HandleFunc("/{_:.*}", h.ServeHTTP).Methods("GET")

	http.Handle("/", r) // Set the router as the default handler.

	klog.Fatal(http.ListenAndServe(":"+h.Port, nil))
}
