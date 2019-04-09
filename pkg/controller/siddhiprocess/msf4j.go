package siddhiprocess

import (
    "net/http"
	"encoding/json"
	"context"
	"errors"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"k8s.io/apimachinery/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// SiddhiApp contains details about the siddhi app which need by K8s
type SiddhiApp struct {
	Name string `json:"appName"`
	Ports []int `json:"ports"`
	Protocols []string `json:"protocols"`
	TLS []bool `json:"tls"`
	App string `json:"app"`
}

func (reconcileSiddhiProcess *ReconcileSiddhiProcess) getSiddhiAppInfo(siddhiProcess *siddhiv1alpha1.SiddhiProcess) (siddhiAppStruct SiddhiApp){
	query := siddhiProcess.Spec.Query
	reqLogger := log.WithValues("Request.Namespace", siddhiProcess.Namespace, "Request.Name", siddhiProcess.Name)
	var err error
	var response *http.Response
	if (query == "") && (len(siddhiProcess.Spec.Apps) > 0) {
		var ports []int
		var protocols []string
		var tls []bool
		for _, siddhiFileConfigMapName := range siddhiProcess.Spec.Apps {
			configMapData := make(map[string]string)
			configMap := &corev1.ConfigMap{}
			reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: siddhiFileConfigMapName, Namespace: siddhiProcess.Namespace}, configMap)
			for siddhiFileName, siddhiFileContent := range configMap.Data{
				var siddhiAppInstance SiddhiApp
				url := "http://siddhi-process-msf4j." + siddhiProcess.Namespace + ".svc.cluster.local:9095/service/getSiddhiAppInfo/" + siddhiFileName
				req, _ := http.NewRequest("GET", url, nil)
				q := req.URL.Query()
				q.Add("siddhiApp", siddhiFileContent)
				req.URL.RawQuery = q.Encode()
				url = req.URL.String()
				response, err = http.Get(url)
				if err != nil {
					reqLogger.Error(err, "Unable to invoke service %s", url)
				} else {
					defer response.Body.Close()
					json.NewDecoder(response.Body).Decode(&siddhiAppInstance)
					for i, port := range siddhiAppInstance.Ports{
						if !isIn(ports, port){
							ports = append(ports, port)
							protocols = append(protocols, siddhiAppInstance.Protocols[i])
							tls = append(tls, siddhiAppInstance.TLS[i])
						}
					}
					configMapData[siddhiFileName] = siddhiAppInstance.App
				}
			}
			reqLogger := log.WithValues("Request.Namespace", siddhiProcess.Namespace, "Request.Name", siddhiProcess.Name)
			configMap = &corev1.ConfigMap{}
			configMapName := siddhiFileConfigMapName + "-siddhi"
			err := reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: configMapName, Namespace: siddhiProcess.Namespace}, configMap)
			if err != nil && apierrors.IsNotFound(err) {
				// Define a new CM
				configMap = reconcileSiddhiProcess.createConfigMap(siddhiProcess, configMapData, configMapName)
				reqLogger.Info("Creating a new CM", "CM.Namespace", configMap.Namespace, "CM.Name", configMap.Name)
				err = reconcileSiddhiProcess.client.Create(context.TODO(), configMap)
				if err != nil {
					reqLogger.Error(err, "Failed to create new CM", "CM.Namespace", configMap.Namespace, "CM.Name", configMap.Name)
				}
			} else if err != nil {
				reqLogger.Error(err, "Failed to get CM")
			}
		}
		siddhiAppStruct = SiddhiApp{
			Name: strings.ToLower(siddhiProcess.Name),
			Ports: ports,
			Protocols: protocols,
			TLS: tls,
		}
	} else if (query != "") && (len(siddhiProcess.Spec.Apps) <= 0) {
		siddhiFileName := getAppName(query)
		url := "http://siddhi-process-msf4j." + siddhiProcess.Namespace + ".svc.cluster.local:9095/service/getSiddhiAppInfo/" + siddhiFileName
		req, _ := http.NewRequest("GET", url, nil)
		q := req.URL.Query()
		q.Add("siddhiApp", query)
		req.URL.RawQuery = q.Encode()
		url = req.URL.String()
		response, err = http.Get(url)
		defer response.Body.Close()
		json.NewDecoder(response.Body).Decode(&siddhiAppStruct)
		siddhiAppStruct.Name = strings.ToLower(siddhiProcess.Name)
	} else if (query != "") && (len(siddhiProcess.Spec.Apps) > 0){
		err = errors.New("CRD should only contain either query or app entry")
	} else {
		err = errors.New("CRD must have either query or app entry to deploy siddhi apps")
	}
	return siddhiAppStruct
}

func isIn(slice []int, element int) (bool){
	for _, e := range slice {
		if e == element{
			return true
		}
	}
	return false
}


// configMapForSiddhiApp returns a config map for the query string specified by the user in CRD
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) createConfigMap(siddhiProcess *siddhiv1alpha1.SiddhiProcess, dataMap map[string]string, configMapName string) *corev1.ConfigMap{
	
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: siddhiProcess.Namespace,
		},
		Data: dataMap,
	}
	controllerutil.SetControllerReference(siddhiProcess, configMap, reconcileSiddhiProcess.scheme)
	return configMap
}