package siddhiprocess

import(
	"regexp"
	"strings"
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	siddhiv1alpha1 "github.com/siddhi-io/siddhi-operator/pkg/apis/siddhi/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// deploymentForMSiddhiProcess returns a siddhiProcess Deployment object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) deploymentForSiddhiProcess(siddhiProcess *siddhiv1alpha1.SiddhiProcess) (*appsv1.Deployment, error) {
	labels := labelsForSiddhiProcess(siddhiProcess.Name)
	reqLogger := log.WithValues("Request.Namespace", siddhiProcess.Namespace, "Request.Name", siddhiProcess.Name)
	replicas := siddhiProcess.Spec.Size
	query := siddhiProcess.Spec.Query
	secrets := siddhiProcess.Spec.Secrets
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	var imagePullSecrets []corev1.LocalObjectReference
	var err error
	if  (query == "") && (len(siddhiProcess.Spec.Apps) > 0) {
		for _, siddhiFileConfigMapName := range siddhiProcess.Spec.Apps {
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
			volumes = append(volumes, volume)
			for siddhiFileNameValue := range configMap.Data{
				volumeMount := corev1.VolumeMount{
					Name: siddhiFileConfigMapName,
					MountPath: "/home/siddhi-runner-1.0.0-SNAPSHOT/wso2/worker/deployment/siddhi-files/" + siddhiFileNameValue,
					SubPath:  siddhiFileNameValue,
				}
				volumeMounts = append(volumeMounts, volumeMount)
			}
		}
	} else if (query != "") && (len(siddhiProcess.Spec.Apps) <= 0){
		query = strings.TrimSpace(query)
		re := regexp.MustCompile(".*@App:name\\(\"(.*)\"\\)")
		match := re.FindStringSubmatch(query)
		appName := match[1]
		configMapName := strings.ToLower(appName)
		configMap := reconcileSiddhiProcess.configMapForSiddhiApp(siddhiProcess, query, appName)
		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		err := reconcileSiddhiProcess.client.Create(context.TODO(), configMap)
		if err != nil {
			reqLogger.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		} else{
			volume := corev1.Volume {
				Name: configMapName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
					},
				},
			}
			volumes = append(volumes, volume)
		
			volumeMount := corev1.VolumeMount{
				Name: configMapName,
				MountPath: "/home/siddhi-runner-1.0.0-SNAPSHOT/wso2/worker/deployment/siddhi-files/" + appName + ".siddhi",
				SubPath:  appName + ".siddhi",
			}
			volumeMounts = append(volumeMounts, volumeMount)
		}
	} else if (query != "") && (len(siddhiProcess.Spec.Apps) > 0){
		err = errors.New("CRD should only contain either query or app entry")
	} else {
		err = errors.New("CRD must have either query or app entry to deploy siddhi apps")
	}
	if len(secrets) > 0 {
		for _, secret := range secrets{
			localObject := corev1.LocalObjectReference{
				Name: string(secret.Name),
			}
			imagePullSecrets = append(imagePullSecrets, localObject)
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
					ImagePullSecrets: imagePullSecrets,
					Volumes: volumes,
				},
			},
		},
	}
	// Set SiddhiProcess instance as the owner and controller
	controllerutil.SetControllerReference(siddhiProcess, sidddhiDeployment, reconcileSiddhiProcess.scheme)
	return sidddhiDeployment, err
}

// serviceForSiddhi returns a Siddhi Service object
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) configMapForSiddhiApp(siddhiProcess *siddhiv1alpha1.SiddhiProcess, query string, appName string) *corev1.ConfigMap {
	configMapKey := appName + ".siddhi"
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(appName),
			Namespace: siddhiProcess.Namespace,
		},
		Data: map[string]string{
			configMapKey: query,
		},
	}
	// Set Siddhi instance as the owner and controller
	controllerutil.SetControllerReference(siddhiProcess, configMap, reconcileSiddhiProcess.scheme)
	return configMap
}