package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"time"

	v1 "k8s.io/api/core/v1"
)

type ConfigIO struct {
	clientset   *kubernetes.Clientset
	mapping     map[string]uint16
	mappingName string
}

func (k ConfigIO) startListen(updateConfigMap func(map[string]string)) {
	watch, err := k.clientset.CoreV1().ConfigMaps("default").Watch(context.TODO(), metav1.ListOptions{})
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
				configMap, ok := event.Object.(*v1.ConfigMap)
				if !ok {
					continue
				}
				updateConfigMap(configMap.Data)
			}
		case <-time.After(30 * time.Second):
			fmt.Println("timed out")
		}
	}
}

func (k ConfigIO) UpdateMapping() (map[string]string, error) {
	klog.Info("Load config...")
	maps, err := k.clientset.CoreV1().ConfigMaps("default").Get(context.TODO(), k.mappingName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return maps.Data, nil
}
