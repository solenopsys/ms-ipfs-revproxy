package main

import (
	"flag"
	"github.com/joho/godotenv"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"sc-bm-ipfs-revproxy/internal"
	"sc-bm-ipfs-revproxy/pkg/utils"
)

var Mode string

const DEV_MODE = "dev"

func getCubeConfig(devMode bool) (*rest.Config, error) {
	if devMode {
		var kubeconfigFile = os.Getenv("kubeconfigPath")
		kubeConfigPath := filepath.Join(kubeconfigFile)
		klog.Infof("Using kubeconfig: %s\n", kubeConfigPath)

		kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			klog.Error("error getting Kubernetes config: %v\n", err)
			os.Exit(1)
		}

		return kubeConfig, nil
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		return config, nil
	}
}
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

	config, err := getCubeConfig(devMode)
	if err != nil {
		klog.Info("Config init error...", err)
		os.Exit(1)
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		klog.Fatal(err)
	}

	h := &utils.ProxyPool{
		Port:       port,
		HostTarget: map[string]string{},
		HostProxy:  map[string]*utils.ProxyHolder{},
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
