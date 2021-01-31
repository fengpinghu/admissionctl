package main

import (
	//"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"strconv"
	"strings"

	//"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubernetes/pkg/apis/core/v1"

	//"github.com/fengpinghu/admissionctl/pkg/patch"
	clientv1beta1 "github.com/fengpinghu/admissionctl/pkg/client/v1beta1"
	patchv1beta1 "github.com/fengpinghu/admissionctl/pkg/patch/v1beta1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	//metav1.NamespacePublic,
}

type WebhookServer struct {
	admConfig *patchv1beta1.Conf
	server        *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	port           int    // webhook server port
	certFile       string // path to the x509 certificate for https
	keyFile        string // path to the x509 private key matching `CertFile`
	admCfgFile string // path to admission configuration file
}
/*
type Config struct {
	Containers []corev1.Container `yaml:"containers"`
	Volumes    []corev1.Volume    `yaml:"volumes"`
}
*/
func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

func validationRequired(ignoredList []string, name, ns string) bool {
	// skip special kubernete system namespaces
	for _, namespace := range ignoredList {
		if ns == namespace {
			glog.Infof("Skip validation for %v for it' in special namespace:%v", name, ns)
			return false
		}
	}
	required := true
	return required
}

// Check whether the target resoured need to be mutated
func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	// skip special kubernete system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip mutation for %v for it' in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}
	required := true
	return required
}

/*
// use these two function if build with cgo disabled
func getUIDfromName(username string) (uid int64, err error) {

	cmd := exec.Command("id", "-u", username)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		glog.Infof("can't get uid for username=%v,out=%v,err=%v", username, out.String(), err)
	} else {
		uid, err = strconv.ParseInt(strings.TrimSuffix(out.String(), "\n"), 10, 64)
		glog.Infof("uid for username=%v is %v", username, uid)

	}
	return
}

func getGIDfromName(gname string) int64 {

	first := exec.Command("getent", "group", gname)
	second := exec.Command("cut", "-d:", "-f3")

	// http://golang.org/pkg/io/#Pipe

	reader, writer := io.Pipe()

	// push first command output to writer
	first.Stdout = writer

	// read from first command output
	second.Stdin = reader

	// prepare a buffer to capture the output
	// after second command finished executing
	var buff bytes.Buffer
	second.Stdout = &buff

	first.Start()
	second.Start()
	first.Wait()
	writer.Close()
	second.Wait()

	out := buff.String() // convert output to string
	gid, _ := strconv.ParseInt(strings.TrimSuffix(out, "\n"), 10, 64)
	return gid
}
*/

func (whsvr *WebhookServer) validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	allowed := true
	req := ar.Request
	var (
		result *metav1.Status
	)

	glog.Infof("AdmissionReview(validate) for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
	/*
		if !validationRequired(ignoredNamespaces, req.Name, req.Namespace) {
			glog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	*/

	usr, err := user.Lookup(req.UserInfo.Username)
	if err != nil {
		glog.Infof("user lookup for: %v - %v, request allowed", req.UserInfo.Username, err.Error())
		return &v1beta1.AdmissionResponse{
			Allowed: allowed,
			Result:  result,
		}
	}

	//handle deletes
	if req.Operation == "DELETE" {
		uid, err := clientv1beta1.GetUID(req.Kind, req.Name, req.Namespace)
		if err == nil {
			glog.Infof("%v: %v, uid : %d", req.Kind.Kind, req.Name, uid)
		}
		//uid_req, err1 := getUIDfromName(req.UserInfo.Username)
		glog.Infof("request: uid : %v", usr.Uid)
		uid_req, _ := strconv.ParseInt(usr.Uid, 10, 64)
		if (err != nil) || (uid != uid_req) {
			allowed = false
			result = &metav1.Status{
				Reason: "You are not allowed to delete this resource, please contact admin if you have any questions!",
			}
		}
	} else if req.Operation == "CREATE" {
		coninf, err := patchv1beta1.GetConIntf(req)
		if err != nil {
			glog.Errorf("Could not get container interface: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		if coninf == nil {
			glog.Infof("Skipping validation for apigroup:%s/kind:%s ", req.Kind.Group, req.Kind.Kind)
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}

		}

		for k, spec := range coninf.GetPodSpecs() {
			for _, volume := range spec.Volumes {
				glog.Infof("path:%v, Volumes: %v", k, volume.Name)
				if volume.HostPath != nil {
					//glog.Infof("req name: %v,path: %v",req.Name, volume.HostPath.Path )
					found := false
					//for _, v := range *allowedVolumes {
					for _, v := range whsvr.admConfig.AllowedFileSystem {
						if strings.HasPrefix(volume.HostPath.Path, v) {
							found = true
						}
					}
					if found == false {
						glog.Infof("Path:%v is not allowed", volume.HostPath.Path)
						allowed = false
						//var reason metav1.StatusReason
						msg := fmt.Sprintf("You are not allowed to acess hostpath: %v", volume.HostPath.Path)
						result = &metav1.Status{
							Reason: metav1.StatusReason(msg),
						}
					}
				}
			}
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: allowed,
		Result:  result,
	}
}

func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v, Name=%v, UID=%v, patchOperation=%v, UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
	// determine whether to perform mutation
	/*
		if !mutationRequired(ignoredNamespaces, &pod.ObjectMeta) {
			glog.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	*/

	var gids []int64
	var uid int64
	//uid, err := getUIDfromName(req.UserInfo.Username)
	if usr, err := user.Lookup(req.UserInfo.Username); err != nil {
		glog.Infof("Skipping mutation for user: %v", req.UserInfo.Username)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	} else {
		uid, _ = strconv.ParseInt(usr.Uid, 10, 64)
		sgids, _ := usr.GroupIds()
		for _, v := range sgids {
			id, _ := strconv.ParseInt(v, 10, 64)
			gids = append(gids, id)
		}
		glog.Infof("name: %v, uid=%v, gids=%v", req.UserInfo.Username, usr.Uid, gids)
		//glog.Infof("gids=%v", gids)
	}

	coninf, err := patchv1beta1.GetConIntf(req)
	if err != nil {
		glog.Errorf("Could not get container interface: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}
	if coninf == nil {
		glog.Infof("Skipping mutation for apigroup:%s/kind:%s ", req.Kind.Group, req.Kind.Kind)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	for k, v := range coninf.GetPodSpecs() {
		for _, c := range v.Containers {
			glog.Infof("path:%v, container: %v", k, c.Name)
		}
	}

	var patchBytes []byte
	usercfg := whsvr.admConfig.GetUserCfg(req.UserInfo.Username)
	patchBytes, err = patchv1beta1.CreatePatch(coninf, uid, gids, req.UserInfo.Username, usercfg)
	//patchBytes, err = patchv1beta1.CreatePatch(coninf, uid, gids, req.UserInfo.Username)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
	/*
		var pod corev1.Pod
		//glog.Errorf("raw:%s", req.Object.Raw)
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
			req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)
	*/
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		//admissionResponse = whsvr.mutate(&ar)
		fmt.Println(r.URL.Path)
		if r.URL.Path == "/mutate" {
			admissionResponse = whsvr.mutate(&ar)
		} else if r.URL.Path == "/validate" {
			admissionResponse = whsvr.validate(&ar)
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
