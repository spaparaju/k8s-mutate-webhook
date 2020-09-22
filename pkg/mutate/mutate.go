// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package mutate

import (
	"encoding/json"
	"fmt"
	"strings"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Mutate(body []byte, verbose bool) ([]byte, error) {
	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	var pod *corev1.Pod

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		if len(ar.Namespace) > 0 &&
			strings.HasPrefix(ar.Namespace, "openshift-") { // &&
			//	!strings.HasPrefix(ar.Namespace, "openshift-kube-apiserver") &&
			//	!strings.HasPrefix(ar.Namespace, "openshift-kube-controller-manager") &&
			//	!strings.HasPrefix(ar.Namespace, "openshift-kube-scheduler") &&
			//	!strings.HasPrefix(ar.Namespace, "openshift-etcd") {
			fmt.Printf(" ++++++ Trying to patch  for the namespace : %s\n", ar.Namespace)

			pT := v1beta1.PatchTypeJSONPatch
			resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

			// add some audit annotations, helpful to know why a object was modified, maybe (?)
			resp.AuditAnnotations = map[string]string{
				"crc-mutate-webhook": "initial resource requests been adjusted by crc-mutate-webhook",
			}

			p := []map[string]string{}
			for i := range pod.Spec.Containers {

				minimalCPUValue := getMinimalCPUValue(ar.Namespace)
				minimalMemoryValue := getMinimalMemoryValue(ar.Namespace)

				// Apply minimal Memory requests
				var memoryPatch map[string]string
				currentMemory := pod.Spec.Containers[i].Resources.Requests.Memory().String()
				if currentMemory != "0" {
					memoryPatch = map[string]string{
						"op":    "replace",
						"path":  fmt.Sprintf("/spec/containers/%d/resources/requests/memory", i),
						"value": minimalMemoryValue,
					}
					p = append(p, memoryPatch)
				}

				// Apply minimal CPU requests
				var cpuPatch map[string]string
				currentCPU := pod.Spec.Containers[i].Resources.Requests.Cpu().String()
				if currentCPU != "0" {
					cpuPatch = map[string]string{
						"op":    "replace",
						"path":  fmt.Sprintf("/spec/containers/%d/resources/requests/cpu", i),
						"value": minimalCPUValue,
					}
					p = append(p, cpuPatch)
				}

				// Remove memory limits
				var memoryLimitsPatch map[string]string
				currentMemoryLimits := pod.Spec.Containers[i].Resources.Limits.Memory().String()
				if currentMemoryLimits != "0" {
					memoryLimitsPatch = map[string]string{
						"op":   "remove",
						"path": fmt.Sprintf("/spec/containers/%d/resources/limits/memory", i),
					}
					p = append(p, memoryLimitsPatch)
				}

				// Remove cpu limits
				var cpuLimitsPatch map[string]string
				currentCPULimits := pod.Spec.Containers[i].Resources.Limits.Cpu().String()
				if currentCPULimits != "0" {
					cpuLimitsPatch = map[string]string{
						"op":   "remove",
						"path": fmt.Sprintf("/spec/containers/%d/resources/limits/cpu", i),
					}
					p = append(p, cpuLimitsPatch)
				}

				if memoryPatch != nil || cpuPatch != nil || memoryLimitsPatch != nil || cpuLimitsPatch != nil {
					resp.Patch, err = json.Marshal(p)
				}
			}
			fmt.Printf(" ********** Patched for the namespace : %s\n", ar.Namespace)

		} else {
			fmt.Printf("--------- NOT patching for the namespace : %s\n", ar.Namespace)
		}
		resp.Result = &metav1.Status{
			Status: "Success",
		}
		admReview.Response = &resp
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err
		}
	}
	return responseBody, nil
}

func getMinimalCPUValue(namespace string) string {
	minimalCPUValue := "10m"

	if namespace == "openshift-console" {
		minimalCPUValue = "100m"
	}
	if namespace == "openshift-kube-controller-manager" {
		minimalCPUValue = "300m"
	}
	if namespace == "openshift-kube-apiserver" {
		minimalCPUValue = "800m"
	}
	if namespace == "openshift-etcd" {
		minimalCPUValue = "600m"
	}
	return minimalCPUValue
}

func getMinimalMemoryValue(namespace string) string {
	minimalMemoryValue := "10Mi"
	if namespace == "openshift-console" {
		minimalMemoryValue = "50Mi"
	}
	return minimalMemoryValue
}
