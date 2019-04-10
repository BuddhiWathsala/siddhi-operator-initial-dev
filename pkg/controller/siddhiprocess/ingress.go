package siddhiprocess

import(
	"strings"
	"strconv"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// IntOrString integer or string
type IntOrString struct {
	Type   Type   `protobuf:"varint,1,opt,name=type,casttype=Type"`
	IntVal int32  `protobuf:"varint,2,opt,name=intVal"`
	StrVal string `protobuf:"bytes,3,opt,name=strVal"`
}

// Type represents the stored type of IntOrString.
type Type int

// Int - Type
const (
	Int intstr.Type = iota
	String
)

// loadBalancerForSiddhi returns a Siddhi Ingress load balancer object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) loadBalancerForSiddhiProcess(siddhiProcess *siddhiv1alpha1.SiddhiProcess, siddhiApp SiddhiApp) *extensionsv1beta1.Ingress {
	var ingressPaths []extensionsv1beta1.HTTPIngressPath
	for _, port := range siddhiApp.Ports{
		path := "/" + strings.ToLower(siddhiApp.Name) + "/" + strconv.Itoa(port) + "/"
		ingressPath := extensionsv1beta1.HTTPIngressPath{
			Path: path,
			Backend: extensionsv1beta1.IngressBackend{
				ServiceName: siddhiProcess.Name, 
				ServicePort: intstr.IntOrString{
					Type: Int, 
					IntVal: int32(port),
				},
			},
		}
		ingressPaths = append(ingressPaths, ingressPath)
	}
	var ingressSpec extensionsv1beta1.IngressSpec
	if siddhiProcess.Spec.SiddhiIngress.TLSSpec.SecretName != "" {
		ingressSpec = extensionsv1beta1.IngressSpec{
			TLS: []extensionsv1beta1.IngressTLS{
				extensionsv1beta1.IngressTLS{
					Hosts: []string{"siddhi"},
					SecretName: siddhiProcess.Spec.SiddhiIngress.TLSSpec.SecretName,
				},
			},
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: "siddhi",
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: ingressPaths,
						},
					},
				},
			},
		}
	} else {
		ingressSpec = extensionsv1beta1.IngressSpec{
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: "siddhi",
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: ingressPaths,
						},
					},
				},
			},
		}
	}
	ingress := &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "siddhi",
			Namespace: siddhiProcess.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		Spec: ingressSpec,
	}
	return ingress
}



// updatedLoadBalancerForSiddhiProcess returns a Siddhi Ingress load balancer object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) updatedLoadBalancerForSiddhiProcess(siddhiProcess *siddhiv1alpha1.SiddhiProcess, currentIngress *extensionsv1beta1.Ingress, siddhiApp SiddhiApp) *extensionsv1beta1.Ingress {
	var ingressPaths []extensionsv1beta1.HTTPIngressPath
	for _, port := range siddhiApp.Ports{
		path := "/" + strings.ToLower(siddhiApp.Name) + "/" + strconv.Itoa(port) + "/"
		ingressPath := extensionsv1beta1.HTTPIngressPath{
			Path: path,
			Backend: extensionsv1beta1.IngressBackend{
				ServiceName: siddhiProcess.Name, 
				ServicePort: intstr.IntOrString{
					Type: Int, 
					IntVal: int32(port),
				},
			},
		}
		ingressPaths = append(ingressPaths, ingressPath)
	}
	
	currentRules := currentIngress.Spec.Rules
	newRule := extensionsv1beta1.IngressRule{
		Host: "siddhi",
		IngressRuleValue: extensionsv1beta1.IngressRuleValue{
			HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
				Paths: ingressPaths,
			},
		},
	}
	ruleExists := false
	for _, rule := range currentRules{
		if reflect.DeepEqual(rule, newRule){
			ruleExists = true
		}
	}
	if !ruleExists{
		currentRules = append(currentRules, newRule)
	}
	var ingressSpec extensionsv1beta1.IngressSpec
	if siddhiProcess.Spec.SiddhiIngress.TLSSpec.SecretName != "" {
		ingressSpec = extensionsv1beta1.IngressSpec{
			TLS: []extensionsv1beta1.IngressTLS{
				extensionsv1beta1.IngressTLS{
					Hosts: []string{"siddhi"},
					SecretName: siddhiProcess.Spec.SiddhiIngress.TLSSpec.SecretName,
				},
			},
			Rules: currentRules,
		}
	} else {
		ingressSpec = extensionsv1beta1.IngressSpec{
			Rules: currentRules,
		}
	}
	currentIngress.Spec = ingressSpec
	return currentIngress
}