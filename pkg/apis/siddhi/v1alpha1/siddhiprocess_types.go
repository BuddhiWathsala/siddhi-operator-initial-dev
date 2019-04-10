package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnviromentVariable to store env name and value
type EnviromentVariable struct{
	Name string `json:"name"`
	Value string `json:"value"`
}

// SiddhiIngress contains ingress specs for siddhi 
type SiddhiIngress struct{
	TLSSpec TLS `json:"tls"`
}
// TLS contains the TLS configuration of ingress
type TLS struct{
	SecretName string `json:"secretName"`
}

// SiddhiProcessSpec defines the desired state of SiddhiProcess
// +k8s:openapi-gen=true
type SiddhiProcessSpec struct {
	Apps []string `json:"apps"`
	Query string `json:"query"`
	SiddhiConfig string `json:"siddhi.runner.configs"`
	EnviromentVariables []EnviromentVariable `json:"env"`
	SiddhiIngress SiddhiIngress `json:"ingress"`
}

// SiddhiProcessStatus defines the observed state of SiddhiProcess
// +k8s:openapi-gen=true
type SiddhiProcessStatus struct {
	Nodes []string `json:"nodes"`
}

// SiddhiProcess is the Schema for the siddhiprocesses API
// +k8s:openapi-gen=true
type SiddhiProcess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SiddhiProcessSpec   `json:"spec,omitempty"`
	Status SiddhiProcessStatus `json:"status,omitempty"`
}

// SiddhiProcessList contains a list of SiddhiProcess
type SiddhiProcessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SiddhiProcess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SiddhiProcess{}, &SiddhiProcessList{})
}
