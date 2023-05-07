package internal

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"time"

	v1 "k8s.io/api/core/v1"
)

type ConfigIO struct {
	ClientSet       *kubernetes.Clientset
	UpdateConfigMap func(map[string]string)
	MappingName     string
}

func (conf *ConfigIO) Listen() {
	watch, err := conf.ClientSet.CoreV1().ConfigMaps("default").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	defer watch.Stop()

	for {
		select {
		case event, ok := <-watch.ResultChan():
			if !ok {
				return
			}
			if event.Type == "MODIFIED" {
				klog.Infoln("Modified")
				configMap, ok := event.Object.(*v1.ConfigMap)

				klog.Infoln("Config map name ", configMap.Name)
				if !ok {
					klog.Infoln("Modified not ok")
					continue
				} else if configMap.Name != conf.MappingName {
					conf.UpdateConfigMap(configMap.Data)
				}
			}
		case <-time.After(30 * time.Second):
			klog.Infoln("Next interval")
		}
	}
}

func (conf *ConfigIO) LoadMapping() {
	klog.Info("Load config...")
	maps, err := conf.ClientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), conf.MappingName, metav1.GetOptions{})
	if err != nil {
		klog.Fatal("Error load config: %s", err.Error())
	}

	conf.UpdateConfigMap(maps.Data)
}
