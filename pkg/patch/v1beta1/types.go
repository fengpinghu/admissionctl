package v1beta1

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	vbv1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type MybJob batchv1.Job

type MyDeployment appsv1.Deployment

type MyVCJob vbv1alpha1.Job

type MyPod corev1.Pod

type MyConIntf interface {
	GetPodSpecs() map[string]corev1.PodSpec
	GetTemplates() []string
}
