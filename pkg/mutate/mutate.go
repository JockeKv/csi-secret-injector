package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JSONPatch struct {
	Op    string      `json:"op,omitempty"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type VolumeMount struct {
	Name      string `json:"name,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

type Volume struct {
	Name string   `json:"name,omitempty"`
	Csi  CSIDrive `json:"csi,omitempty"`
}

type CSIDrive struct {
	Driver           string            `json:"driver,omitempty"`
	ReadOnly         bool              `json:"readOnly,omitempty"`
	VolumeAttributes map[string]string `json:"volumeAttributes,omitempty"`
}

// Validate and mutate the rqquest
func Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body))
	}

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

		var secretClass, secretContainer string

		if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID

		if val, ok := pod.Annotations["xcxc.dev/csi-secret-class"]; ok {
			secretClass = val
		}

		if val, ok := pod.Annotations["xcxc.dev/csi-secret-container"]; ok {
			secretContainer = val
		}

		// Variables for control-flow
		hasVolume := false
		hasVolumeMount := false
		container := 0

		// Check if we already have the Volume mounted
		for _, v := range pod.Spec.Volumes {
			if v.Name == "secret-store" {
				hasVolume = true
				break
			}
		}

		// Check if we already have the volume mounted on the desired container
		for i, v := range pod.Spec.Containers {
			if secretContainer != "" && v.Name == secretContainer {
				container = i
			}
		}
		for i, v := range pod.Spec.Containers {
			for _, m := range v.VolumeMounts {
				if m.Name == "secret-store" && container == i {
					hasVolumeMount = true
				}
			}
		}

		// Check for secretClass annotarion and create patch if found
		if secretClass != "" {
			log.Printf("Got pod with secret class: %s, Patching..", secretClass)
			p := []JSONPatch{}

			// Create volume is we don't alreade have it
			if !hasVolume {
				volume := JSONPatch{
					Op:   "add",
					Path: "/spec/volumes/-",
					Value: Volume{
						Name: "secret-store",
						Csi: CSIDrive{
							Driver:   "secrets-store.csi.k8s.io",
							ReadOnly: true,
							VolumeAttributes: map[string]string{
								"secretProviderClass": secretClass,
							},
						},
					},
				}
				p = append(p, volume)
			}

			// Create volumeMount if we don't already have it
			if !hasVolumeMount {
				volumeMount := JSONPatch{
					Op: "add",
					// "path": []byte(fmt.Sprintf("/spec/containers/%d/volumeMounts/-", secretContainer)),
					Path: fmt.Sprintf("/spec/containers/%d/volumeMounts/-", container),
					Value: VolumeMount{
						Name:      "secret-store",
						MountPath: "/mnt/secret-store",
						ReadOnly:  true,
					},
				}
				p = append(p, volumeMount)
			}

			// If we have anything to patch; marshal it to json and add to response
			if len(p) > 0 {
				// parse the []map into JSON
				resp.Patch, err = json.Marshal(p)
				if err != nil {
					return nil, err
				}

				log.Printf("Patch: %s\n", resp.Patch)

				// Set PatchType
				pT := v1beta1.PatchTypeJSONPatch
				resp.PatchType = &pT
			} else {
				log.Println("Nothing to patch.")
			}

			resp.Result = &metav1.Status{
				Status: "Success",
			}

		} else {
			log.Println("No patching needed.")
		}

		admReview.Response = &resp

		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody))
	}

	return responseBody, nil
}
