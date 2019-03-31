package siddhiprocess

import(
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// loadBalancerForSiddhi returns a Siddhi Ingress load balancer object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) loadBalancerForSiddhiProcess(m *siddhiv1alpha1.SiddhiProcess) *extensionsv1beta1.Ingress {
	ingress := &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
				"nginx.ingress.kubernetes.io/ssl-redirect": "false",
				"nginx.ingress.kubernetes.io/force-ssl-redirect": "false",
				"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
				"nginx.ingress.kubernetes.io/affinity": "cookie",
				"nginx.ingress.kubernetes.io/session-cookie-name": "route",
				"nginx.ingress.kubernetes.io/session-cookie-hash": "sha1",
			},
		},
		Spec: extensionsv1beta1.IngressSpec{
			Rules: []extensionsv1beta1.IngressRule{
				{
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path:    "/",
									Backend: extensionsv1beta1.IngressBackend{ServiceName: m.Name, ServicePort: intstr.IntOrString{Type: Int, IntVal: 8006}},
								},
							},
						},
					},
				},
			},
		},
	}
	// Set Siddhi instance as the owner and controller
	controllerutil.SetControllerReference(m, ingress, reconcileSiddhiProcess.scheme)
	return ingress
}