package siddhiprocess

import(
	"strconv"
	"strings"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// serviceForSiddhi returns a Siddhi Service object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) serviceForSiddhiProcess(siddhiProcess *siddhiv1alpha1.SiddhiProcess, siddhiApp SiddhiApp, operatorEnvs map[string]string) *corev1.Service {
	labels := labelsForSiddhiProcess(siddhiProcess.Name, operatorEnvs)
	var servicePorts []corev1.ServicePort
	for _, port := range siddhiApp.Ports{
		servicePort := corev1.ServicePort{
			Port: int32(port),
			Name: strings.ToLower(siddhiApp.Name) + strconv.Itoa(port),
		}
		servicePorts = append(servicePorts, servicePort)
	}
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      siddhiProcess.Name,
			Namespace: siddhiProcess.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: servicePorts,
			Type: "ClusterIP",
		},
	}
	controllerutil.SetControllerReference(siddhiProcess, service, reconcileSiddhiProcess.scheme)
	return service
}