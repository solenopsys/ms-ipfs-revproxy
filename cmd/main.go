package main

import (
	"flag"
	"github.com/joho/godotenv"
	"github.com/solenopsys/bl-kubernetes-tools"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"os"
	"sc-bm-ipfs-revproxy/internal"
	"sc-bm-ipfs-revproxy/pkg/utils"
	"strings"
)

var Mode string

const DEV_MODE = "dev"

func init() {
	flag.StringVar(&Mode, "mode", "", "a string var")
}

func main() {
	flag.Parse()
	devMode := Mode == DEV_MODE

	if devMode {
		err := godotenv.Load("configs/local.env")
		if err != nil {
			klog.Fatal("Error loading .env file")
		}
	}

	port := os.Getenv("server.Port")

	config, err := bl_kubernetes_tools.GetCubeConfig(devMode)
	if err != nil {
		klog.Info("Config init error...", err)
		os.Exit(1)
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		klog.Fatal(err)
	}

	backsString := os.Getenv("ipfs.Backs")
	hosts := strings.Split(backsString, " ")

	h := &utils.ProxyPool{
		Port:       port,
		HostTarget: map[string]string{},
		HostProxy:  map[string]*utils.ProxyHolder{},
		IpfsHosts:  hosts,
	}

	io := &internal.ConfigIO{
		MappingName: "reverse-proxy-mapping",
		UpdateConfigMap: func(m map[string]string) {
			klog.Info("Config updated...", m)
			h.HostTarget = m
		},
		ClientSet: clientSet,
	}
	io.LoadMapping()
	go io.Listen()

	h.Start()
}
