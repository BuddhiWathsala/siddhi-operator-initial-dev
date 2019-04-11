package siddhiprocess

import (
    "net/http"
	"encoding/json"
	"context"
	"errors"
	"strings"
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"k8s.io/apimachinery/pkg/types"
)

// SiddhiApp contains details about the siddhi app which need by K8s
type SiddhiApp struct {
	Name string `json:"appName"`
	Ports []int `json:"ports"`
	Protocols []string `json:"protocols"`
	TLS []bool `json:"tls"`
	App string `json:"app"`
}

// TemplatedApp contains the templated siddhi app and relevant properties to pass into the parser service
type TemplatedApp struct {
	App string `json:"siddhiApp"`
	PropertyMap map[string]string `json:"propertyMap"`
}

type SiddhiParserRequest struct{
	SiddhiApps []string `json:"siddhiApps"`
	PropertyMap map[string]string `json:"propertyMap"`
}

type SourceDeploymentConfig struct{
	ServiceProtocol string `json:"serviceProtocol"`
	Secured bool `json:"secured"`
	Port int `json:"port"`
}

type SourceList struct{
	SourceDeploymentConfigs []SourceDeploymentConfig `json:"sourceDeploymentConfigs"`
}

type SiddhiAppConfig struct{
	SiddhiApp string `json:"siddhiApp"`
	SiddhiSourceList SourceList `json:"sourceList"`
}

type SiddhiParserResponse struct{
	AppConfig []SiddhiAppConfig `json:"siddhiAppConfigs"`
}

// parseSiddhiApp call MSF4J service and parse a given siddhiApp
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) parseSiddhiApp(siddhiProcess *siddhiv1alpha1.SiddhiProcess) (siddhiAppStruct SiddhiApp, err error){
	query := siddhiProcess.Spec.Query
	reqLogger := log.WithValues("Request.Namespace", siddhiProcess.Namespace, "Request.Name", siddhiProcess.Name)
	var resp *http.Response
	var ports []int
	var protocols []string
	var tls []bool
	configMapData := make(map[string]string)
	if (query == "") && (len(siddhiProcess.Spec.Apps) > 0) {
		var siddhiApps []string
		for _, siddhiFileConfigMapName := range siddhiProcess.Spec.Apps {
			configMap := &corev1.ConfigMap{}
			reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: siddhiFileConfigMapName, Namespace: siddhiProcess.Namespace}, configMap)
			for _, siddhiFileContent := range configMap.Data{
				siddhiApps = append(siddhiApps, siddhiFileContent)
			}
		}
		propertyMap := reconcileSiddhiProcess.populateUserEnvs(siddhiProcess)
		siddhiParserRequest := SiddhiParserRequest{
			SiddhiApps: siddhiApps,
			PropertyMap: propertyMap,
		}
		url := "http://siddhi-parser." + siddhiProcess.Namespace + ".svc.cluster.local:9090/service/query/"
		var siddhiParserResponse SiddhiParserResponse
		b, err := json.Marshal(siddhiParserRequest)
		if err != nil {
			fmt.Println(err)
			return siddhiAppStruct, err
		}
		var jsonStr = []byte(string(b))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			reqLogger.Error(err, "REST invoking error")
			return siddhiAppStruct, err
		}
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&siddhiParserResponse)
		for _, siddhiApp := range siddhiParserResponse.AppConfig{
			app := siddhiApp.SiddhiApp
			appName := strings.TrimSpace(getAppName(app)) + ".siddhi"
			for _, deploymentConf := range siddhiApp.SiddhiSourceList.SourceDeploymentConfigs{
				ports = append(ports, deploymentConf.Port)
				protocols = append(protocols, deploymentConf.ServiceProtocol)
				tls = append(tls, deploymentConf.Secured)
			}
			configMapData[appName] = app	
		}
		configMap := &corev1.ConfigMap{}
		configMapName := strings.ToLower(siddhiProcess.Name) + "-siddhi"
		err = reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: configMapName, Namespace: siddhiProcess.Namespace}, configMap)
		if err != nil && apierrors.IsNotFound(err) {
			configMap = reconcileSiddhiProcess.createConfigMap(siddhiProcess, configMapData, configMapName)
			reqLogger.Info("Creating a new CM", "CM.Namespace", configMap.Namespace, "CM.Name", configMap.Name)
			err = reconcileSiddhiProcess.client.Create(context.TODO(), configMap)
			if err != nil {
				reqLogger.Error(err, "Failed to create new CM", "CM.Namespace", configMap.Namespace, "CM.Name", configMap.Name)
			}
		} else if err != nil {
			reqLogger.Error(err, "Failed to get CM")
		}
		siddhiAppStruct = SiddhiApp{
			Name: strings.ToLower(siddhiProcess.Name),
			Ports: ports,
			Protocols: protocols,
			TLS: tls,
		}
		fmt.Println("App Struct")
		fmt.Println(siddhiAppStruct)
	} else if (query != "") && (len(siddhiProcess.Spec.Apps) <= 0) {
		propertyMap := reconcileSiddhiProcess.populateUserEnvs(siddhiProcess)
		url := "http://siddhi-parser." + siddhiProcess.Namespace + ".svc.cluster.local:9090/service/query/"
		var siddhiParserResponse SiddhiParserResponse
		siddhiParserRequest := SiddhiParserRequest{
			SiddhiApps: []string{query},
			PropertyMap: propertyMap,
		}
		b, err := json.Marshal(siddhiParserRequest)
		if err != nil {
			fmt.Println(err)
			return siddhiAppStruct, err
		}
		var jsonStr = []byte(string(b))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			reqLogger.Error(err, "REST invoking error")
			return siddhiAppStruct, err
		}
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&siddhiParserResponse)
		for _, siddhiApp := range siddhiParserResponse.AppConfig{
			app := siddhiApp.SiddhiApp
			appName := strings.TrimSpace(getAppName(app)) + ".siddhi"
			for _, deploymentConf := range siddhiApp.SiddhiSourceList.SourceDeploymentConfigs{
				ports = append(ports, deploymentConf.Port)
				protocols = append(protocols, deploymentConf.ServiceProtocol)
				tls = append(tls, deploymentConf.Secured)
			}
			configMapData[appName] = app	
		}
		siddhiAppStruct = SiddhiApp{
			Name: strings.ToLower(siddhiProcess.Name),
			Ports: ports,
			Protocols: protocols,
			TLS: tls,
		}
		fmt.Println("Query Struct")
		fmt.Println(siddhiAppStruct)
	} else if (query != "") && (len(siddhiProcess.Spec.Apps) > 0){
		err = errors.New("CRD should only contain either query or app entry")
	} else {
		err = errors.New("CRD must have either query or app entry to deploy siddhi apps")
	}
	return siddhiAppStruct, err
}

// isIn used to find element in a given slice
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


// populateUserEnvs returns a map for the ENVs in CRD
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) populateUserEnvs(siddhiProcess *siddhiv1alpha1.SiddhiProcess) (envs map[string]string){
	envs = make(map[string]string)
	envStruct := siddhiProcess.Spec.EnviromentVariables
	for _, env := range envStruct {
		envs[env.Name] = env.Value
	}
	
	return envs
}