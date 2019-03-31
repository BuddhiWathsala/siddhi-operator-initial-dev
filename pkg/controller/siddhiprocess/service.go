package siddhiprocess

import(
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// serviceForSiddhi returns a Siddhi Service object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) serviceForSiddhiProcess(m *siddhiv1alpha1.SiddhiProcess) *corev1.Service {
	labels := labelsForSiddhiProcess(m.Name)
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Name: "passthrough", Port: 8006, Protocol: "TCP"},
			},
			Type: "LoadBalancer",
		},
	}
	// Set Siddhi instance as the owner and controller
	controllerutil.SetControllerReference(m, service, reconcileSiddhiProcess.scheme)
	return service
}