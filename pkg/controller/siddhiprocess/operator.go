package siddhiprocess

import (
	appsv1 "k8s.io/api/apps/v1"
)

// populateOperatorEnvs returns a map of ENVs in the operator deployment
func (reconcileSiddhiProcess *ReconcileSiddhiProcess) populateOperatorEnvs(operatorDeployment *appsv1.Deployment) (envs map[string]string){
	envs = make(map[string]string)
	envStruct := operatorDeployment.Spec.Template.Spec.Containers[0].Env
	for _, env := range envStruct {
		envs[env.Name] = env.Value
	}
	
	return envs
}

