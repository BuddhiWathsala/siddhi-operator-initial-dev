package siddhiprocess

import (
    "net/http"
    "encoding/json"
)

// SiddhiApp contains details about the siddhi app which need by K8s
type SiddhiApp struct {
	Name string `json:"appName"`
	Ports []int `json:"ports"`
	Protocols []string `json:"protocols"`
	TLS []bool `json:"tls"`
}

func getSiddhiAppInfo() (target []SiddhiApp){
    response, err := http.Get("http://192.168.99.155:32001/service/siddhi")
    if err != nil {
        return target
    }
	defer response.Body.Close()
	json.NewDecoder(response.Body).Decode(&target)
	return target
}