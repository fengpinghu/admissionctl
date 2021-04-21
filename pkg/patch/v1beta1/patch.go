/*
Package v1beta1 is the v1beta1 version of the kubernetes obj patch library. The patch library defines a interface for kubernetes objs and aims to control securitycontext of containers, label pod with the owner, control walltime of the jobs, and etc.

*/

package v1beta1

import (
	"encoding/json"
	"fmt"
	"strconv"
	//"reflect"
	//"strings"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	vbv1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (v MyPod) GetPodSpecs() map[string]corev1.PodSpec {
	//return v.Spec.Containers
	var m map[string]corev1.PodSpec
	m = make(map[string]corev1.PodSpec)
	m["/spec/"] = v.Spec
	return m
}

func (v MyPod) GetTemplates() []string {
	return []string{"/"}
}

func (v MybJob) GetPodSpecs() map[string]corev1.PodSpec {
	//return v.Spec.Template.Spec.Containers
	var m map[string]corev1.PodSpec
	m = make(map[string]corev1.PodSpec)
	m["/spec/template/spec/"] = v.Spec.Template.Spec
	return m
}

func (v MybJob) GetTemplates() []string {
	return []string{"/spec/template/"}
}

func (v MyDeployment) GetPodSpecs() map[string]corev1.PodSpec {
	//return v.Spec.Template.Spec.Containers
	var m map[string]corev1.PodSpec
	m = make(map[string]corev1.PodSpec)
	m["/spec/template/spec/"] = v.Spec.Template.Spec
	return m
}

func (v MyDeployment) GetTemplates() []string {
	return []string{"/spec/template/"}
}

func (v MyVCJob) GetPodSpecs() map[string]corev1.PodSpec {
	var m map[string]corev1.PodSpec
	m = make(map[string]corev1.PodSpec)
	for i, t := range v.Spec.Tasks {
		path := fmt.Sprintf("/spec/tasks/%d/template/spec/", i)
		//for _, c := range t.Template.Spec.Containers {
		//	m[path] = c
		//}
		m[path] = t.Template.Spec
	}
	return m
}

func (v MyVCJob) GetTemplates() []string {
	var tempPaths []string
	for i := 0; i < len(v.Spec.Tasks); i++ {
		tempPaths = append(tempPaths, fmt.Sprintf("/spec/tasks/%d/template/", i))
	}
	return tempPaths
}

//GetConIntf gets kubernetes objects interface that can be used by mutation and validation functions
func GetConIntf(req *v1beta1.AdmissionRequest) (conintf MyConIntf, err error) {
	conintf = nil
	err = nil
	switch req.Kind.Group {
	case "batch.volcano.sh":
		switch req.Kind.Kind {
		case "Job":
			var job vbv1alpha1.Job
			var myjob MyVCJob
			if err = json.Unmarshal(req.Object.Raw, &job); err != nil {
				glog.Errorf("Could not unmarshal raw object: %v", err)
			} else {
				/*glog.Infof("job:%s", job.Name)
				  for _, c := range job.Spec.Template.Spec.Containers {
				          glog.Infof("c name:%s",  c.Name)
				  }*/
				myjob = MyVCJob(job)
				conintf = myjob
			}
		default:
			glog.Infof("ingnore kind from volcano: %v", req.Kind.Kind)
		}
	default:
		switch req.Kind.Kind {
		case "Deployment":
			var deployment appsv1.Deployment
			var mydeployment MyDeployment
			if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
				glog.Errorf("Could not unmarshal raw object: %v", err)
			} else {
				//glog.Infof("deployment:%s", deployment.Name)
				mydeployment = MyDeployment(deployment)
				conintf = mydeployment
			}
		case "Job":
			var job batchv1.Job
			var myjob MybJob
			if err = json.Unmarshal(req.Object.Raw, &job); err != nil {
				glog.Errorf("Could not unmarshal raw object: %v", err)
			} else {
				/*glog.Infof("job:%s", job.Name)
				  for _, c := range job.Spec.Template.Spec.Containers {
				          glog.Infof("c name:%s",  c.Name)
				  }*/
				myjob = MybJob(job)
				conintf = myjob
			}
		case "Pod":
			var pod corev1.Pod
			if err = json.Unmarshal(req.Object.Raw, &pod); err != nil {
				glog.Errorf("Could not unmarshal raw object: %v", err)
			}
			mypod := MyPod(pod)
			conintf = mypod
		default:
			glog.Infof("ingnore kind: %v", req.Kind.Kind)
		}
	}
	return
}

// CreatePatch creates mutation patchs
func CreatePatch(contI MyConIntf, uid int64, gids []int64, username string, usercfg UserCfg) ([]byte, error) {
	var patch []patchOperation

	glog.Infof("usercfg:%+v\n", usercfg)
	//patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)
	patch = append(patch, updateSecurityContext(contI, uid, gids)...)
	patch = append(patch, updateLabel(contI, username)...)
	patch = append(patch, updateDeadline(contI, usercfg)...)

	return json.Marshal(patch)
}

// updateSecurityContext updates/removes securitycontexts 
func updateSecurityContext(contI MyConIntf,
	uid int64,
	gids []int64) (patch []patchOperation) {

	glog.Infof("template path:\n")
	for _, tp_path := range contI.GetTemplates() {
		glog.Infof(tp_path)
		patch = append(patch, patchOperation{
			Op:   "add",
			Path: tp_path + "spec/" + "securityContext",
			Value: map[string]interface{}{"runAsUser": uid,
				"runAsGroup":         100,
				"supplementalGroups": gids,
			},
		})
	}

	for path, spec := range contI.GetPodSpecs() {
		for _, c := range spec.Containers {
			glog.Infof("path:%v, container: %v", path, c.Name)
		}
		for i, c := range spec.Containers {
			if c.SecurityContext != nil {
				glog.Infof("container securitycontext removed for container: %s", c.Name)
				patch = append(patch, patchOperation{
					Op:   "remove",
					Path: path + "containers/" + strconv.Itoa(i) + "/securityContext",
				})
			}
		}
	}
	return patch
}
// updateLabel puts a user=username label on pods
func updateLabel(contI MyConIntf, username string) (patch []patchOperation) {
	//var path string
	//k8s.io/apimachinery/pkg/apis/meta/v1/group_version.go

	glog.Infof("template path:\n")
	for _, tp_path := range contI.GetTemplates() {
		glog.Infof(tp_path)
		patch = append(patch, patchOperation{
			Op:   "add",
			Path: tp_path + "metadata/" + "labels",
			Value: map[string]string{
				"user": username,
			},
		})
	}
	return patch
}

//*
func updateDeadline(contI MyConIntf, usercfg UserCfg) (patch []patchOperation) {

	//xType := reflect.TypeOf(coninf)
	//xValue := reflect.ValueOf(coninf)
	//glog.Infof("type: %v", xType)
	//apply a different walltime limit when user creates pod directly
	var maxWallTime int64
	if _, ok := contI.(MyPod); ok {
		maxWallTime = usercfg.MaxWallTimePod
	} else {
		maxWallTime = usercfg.MaxWallTime
	}

	if maxWallTime == 0 {
		return nil
	}
        for path, _ := range contI.GetPodSpecs() {

                patch = append(patch, patchOperation{
                                Op:    "add",
                                Path:  path + "activeDeadlineSeconds",
                                Value: maxWallTime,
                })
                patch = append(patch, patchOperation{
                                Op:    "add",
                                Path:  path + "restartPolicy",
                                Value: "Never",
                })
        }

        return patch
}

//*/
