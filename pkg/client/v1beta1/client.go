package client

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	//"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
        vcclient "volcano.sh/volcano/pkg/client/clientset/versioned"

)


//GetUID extracts uid information from the RunAsUser field of various kind of kubernetes objects 
func GetUID(kind metav1.GroupVersionKind, name, namespace string) (uid int64, err error) {

	config, err := clientcmd.BuildConfigFromFlags("", "/etc/kubernetes/admin.conf")
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

        switch kind.Group {
        case "batch.volcano.sh":
                switch kind.Kind {
                case "Job":
			jobClient := vcclient.NewForConfigOrDie(config)
			job, err := jobClient.BatchV1alpha1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("failed to get vcjob : %v\n", name)
				err = fmt.Errorf("vcjob not found - name: %v, ns: %v\n", name, namespace)
			} else if (job.Spec.Tasks[0].Template.Spec.SecurityContext != nil) &&
				((job.Spec.Tasks[0].Template.Spec.SecurityContext).RunAsUser != nil) {
				uid = *(job.Spec.Tasks[0].Template.Spec.SecurityContext).RunAsUser
			} else {
				err = fmt.Errorf("uid  for vcjob not found - name: %v, ns: %v\n", name, namespace)
			}
                default:
                        glog.Infof("ingnore kind from volcano: %v", kind.Kind)
			err = fmt.Errorf("ignore kind from volcano: %v", kind.Kind)
		}
	default:
		switch kind.Kind {
		case "Deployment":
			deploymentsClient := clientset.AppsV1().Deployments(namespace)
			dep, err := deploymentsClient.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				err = fmt.Errorf("deployment not found - name: %v, ns: %v\n", name, namespace)
			} else if ((dep.Spec.Template.Spec.SecurityContext) != nil) &&
				((*dep.Spec.Template.Spec.SecurityContext).RunAsUser != nil) {
				uid = *(*(dep.Spec.Template.Spec.SecurityContext)).RunAsUser
			} else {
				err = fmt.Errorf("uid for deployment not found - name: %v, ns: %v\n", name, namespace)
			}
		case "Job":
			jobsClient := clientset.BatchV1().Jobs(namespace)
			job, err := jobsClient.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				err = fmt.Errorf("job not found - name: %v, ns: %v\n", name, namespace)
			} else if ((job.Spec.Template.Spec.SecurityContext) != nil) &&
				((*job.Spec.Template.Spec.SecurityContext).RunAsUser != nil) {
				uid = *(*(job.Spec.Template.Spec.SecurityContext)).RunAsUser
			} else {
				err = fmt.Errorf("uid for job not found - name: %v, ns: %v\n", name, namespace)
			}
		case "Pod":
			podsClient := clientset.CoreV1().Pods(namespace)
			pod, err := podsClient.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				err = fmt.Errorf("pod not found - name: %v, ns: %v\n", name, namespace)
			} else if ((pod.Spec.SecurityContext) != nil) &&
				((*pod.Spec.SecurityContext).RunAsUser != nil) {
				uid = *((pod.Spec.SecurityContext)).RunAsUser
			} else {
				err = fmt.Errorf("uid for pod not found - name: %v, ns: %v\n", name, namespace)
			}
		default:
			err = fmt.Errorf("ignore kind %v", kind.Kind)
		}
	}
	return
}

