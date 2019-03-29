package siddhiprocess

import (
	"context"
	"reflect"

	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_siddhiprocess")

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

// Add creates a new SiddhiProcess Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSiddhiProcess{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("siddhiprocess-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SiddhiProcess
	err = c.Watch(&source.Kind{Type: &siddhiv1alpha1.SiddhiProcess{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner SiddhiProcess
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &siddhiv1alpha1.SiddhiProcess{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSiddhiProcess{}

// ReconcileSiddhiProcess reconciles a SiddhiProcess object
type ReconcileSiddhiProcess struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SiddhiProcess object and makes changes based on the state read
// and what is in the SiddhiProcess.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SiddhiProcess")

	// Fetch the SiddhiProcess instance
	siddhiProcess := &siddhiv1alpha1.SiddhiProcess{}
	err := reconcileSiddhiProcess.client.Get(context.TODO(), request.NamespacedName, siddhiProcess)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: siddhiProcess.Name, Namespace: siddhiProcess.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		siddhiDeployment := reconcileSiddhiProcess.deploymentForSiddhiProcess(siddhiProcess)
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", siddhiDeployment.Namespace, "Deployment.Name", siddhiDeployment.Name)
		err = reconcileSiddhiProcess.client.Create(context.TODO(), siddhiDeployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", siddhiDeployment.Namespace, "Deployment.Name", siddhiDeployment.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := siddhiProcess.Spec.Size
	if *deployment.Spec.Replicas != size {
		deployment.Spec.Replicas = &size
		err = reconcileSiddhiProcess.client.Update(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	service := &corev1.Service{}
	err = reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: siddhiProcess.Name, Namespace: siddhiProcess.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		siddhiService := reconcileSiddhiProcess.serviceForSiddhiProcess(siddhiProcess)
		reqLogger.Info("Creating a new Service", "Service.Namespace", siddhiService.Namespace, "Service.Name", siddhiService.Name)
		err = reconcileSiddhiProcess.client.Create(context.TODO(), siddhiService)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service", "Service.Namespace", siddhiService.Namespace, "Service.Name", siddhiService.Name)
			return reconcile.Result{}, err
		}
		// Service created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service")
		return reconcile.Result{}, err
	}

	ingress := &extensionsv1beta1.Ingress{}
	err = reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: siddhiProcess.Name, Namespace: siddhiProcess.Namespace}, ingress)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Ingress
		siddhiIngress := reconcileSiddhiProcess.loadBalancerForSiddhiProcess(siddhiProcess)
		reqLogger.Info("Creating a new Ingress", "Ingress.Namespace", siddhiIngress.Namespace, "Ingress.Name", siddhiIngress.Name)
		err = reconcileSiddhiProcess.client.Create(context.TODO(), siddhiIngress)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Ingress", "Ingress.Namespace", siddhiIngress.Namespace, "Ingress.Name", siddhiIngress.Name)
			return reconcile.Result{}, err
		}
		// Ingress created successfully - return and requeue
		reqLogger.Info("Ingress created successfully")
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Ingress")
		return reconcile.Result{}, err
	}

	// Update the SiddhiProcess status with the pod names
	// List the pods for this siddhiProcess's deployment
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForSiddhiProcess(siddhiProcess.Name))
	listOps := &client.ListOptions{Namespace: siddhiProcess.Namespace, LabelSelector: labelSelector}
	err = reconcileSiddhiProcess.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "SiddhiProcess.Namespace", siddhiProcess.Namespace, "SiddhiProcess.Name", siddhiProcess.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, siddhiProcess.Status.Nodes) {
		siddhiProcess.Status.Nodes = podNames
		err := reconcileSiddhiProcess.client.Status().Update(context.TODO(), siddhiProcess)
		if err != nil {
			reqLogger.Error(err, "Failed to update SiddhiProcess status")
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

// deploymentForMSiddhiProcess returns a siddhiProcess Deployment object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) deploymentForSiddhiProcess(siddhiProcess *siddhiv1alpha1.SiddhiProcess) *appsv1.Deployment {
	labels := labelsForSiddhiProcess(siddhiProcess.Name)
	replicas := siddhiProcess.Spec.Size
	numberOfConfigMaps := len(siddhiProcess.Spec.Apps)
	volumes := make([]corev1.Volume, numberOfConfigMaps)
	var volumeMounts []corev1.VolumeMount
	for i, siddhiFileConfigMapName := range siddhiProcess.Spec.Apps {
		configMap := &corev1.ConfigMap{}
		reconcileSiddhiProcess.client.Get(context.TODO(), types.NamespacedName{Name: siddhiFileConfigMapName, Namespace: siddhiProcess.Namespace}, configMap)
		volume := corev1.Volume {
			Name: siddhiFileConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: siddhiFileConfigMapName,
					},
				},
			},
		}
		volumes[i] = volume
		for siddhiFileNameValue := range configMap.Data{
			volumeMount := corev1.VolumeMount{
				Name: siddhiFileConfigMapName,
				MountPath: "/home/siddhi-runner-1.0.0-SNAPSHOT/wso2/worker/deployment/siddhi-files/" + siddhiFileNameValue,
				SubPath:  siddhiFileNameValue,
			}
			volumeMounts = append(volumeMounts, volumeMount)
		}
	}

	sidddhiDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      siddhiProcess.Name,
			Namespace: siddhiProcess.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "buddhiwathsala/siddhirunner:v0.0.6",
							Name:  "siddhirunner-runtime",
							Command: []string{
								"sh",
							},
							Args: []string{
								"/home/siddhi-runner-1.0.0-SNAPSHOT/bin/worker.sh",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8006,
									Name: "passthrough",
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
	// Set SiddhiProcess instance as the owner and controller
	controllerutil.SetControllerReference(siddhiProcess, sidddhiDeployment, reconcileSiddhiProcess.scheme)
	return sidddhiDeployment
}

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

// labelsForSiddhiProcess returns the labels for selecting the resources
// belonging to the given siddhiProcess CR name.
func labelsForSiddhiProcess(appName string) map[string]string {
	return map[string]string{
		"name": "SiddhiProcess",
		"instance": appName,
		"version": "1.0.0",
		"part-of": appName,
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}