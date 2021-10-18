package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type admissionData struct {
	ctx        context.Context
	client     ctrlruntimeclient.Client
	seedClient ctrlruntimeclient.Client
}

var jsonPatch = admissionv1.PatchTypeJSONPatch

func New(listenAddress string, client ctrlruntimeclient.Client) (*http.Server, error) {
	mux := http.NewServeMux()
	ad := &admissionData{
		client: client,
	}

	mux.HandleFunc("/machinedeployments", handleFuncFactory(ad.mutateMachineDeployments))
	mux.HandleFunc("/healthz", healthZHandler)

	return &http.Server{
		Addr:    listenAddress,
		Handler: http.TimeoutHandler(mux, 25*time.Second, "timeout"),
	}, nil
}

func healthZHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func createAdmissionResponse(original, mutated runtime.Object) (*admissionv1.AdmissionResponse, error) {
	response := &admissionv1.AdmissionResponse{}
	response.Allowed = true
	if !apiequality.Semantic.DeepEqual(original, mutated) {
		patchOpts, err := newJSONPatch(original, mutated)
		if err != nil {
			return nil, fmt.Errorf("failed to create json patch: %v", err)
		}

		patchRaw, err := json.Marshal(patchOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal json patch: %v", err)
		}
		klog.V(3).Infof("Produced jsonpatch: %s", string(patchRaw))

		response.Patch = patchRaw
		response.PatchType = &jsonPatch
	}
	return response, nil
}

func newJSONPatch(original, current runtime.Object) ([]jsonpatch.JsonPatchOperation, error) {
	originalGVK := original.GetObjectKind().GroupVersionKind()
	currentGVK := current.GetObjectKind().GroupVersionKind()
	if !reflect.DeepEqual(originalGVK, currentGVK) {
		return nil, fmt.Errorf("GroupVersionKind %#v is expected to match %#v", originalGVK, currentGVK)
	}
	ori, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	klog.V(6).Infof("jsonpatch: Marshaled original: %s", string(ori))
	cur, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	klog.V(6).Infof("jsonpatch: Marshaled target: %s", string(cur))
	return jsonpatch.CreatePatch(ori, cur)
}

type mutator func(admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)

func handleFuncFactory(mutate mutator) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		review, err := readReview(r)
		if err != nil {
			klog.Warningf("invalid admission review: %v", err)

			// proper AdmissionReview responses require metadata that is not available
			// in broken requests, so we return a basic failure response
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(fmt.Sprintf("invalid request: %v", err))); err != nil {
				klog.Errorf("failed to write badRequest: %v", err)
			}
			return
		}

		// run the mutation logic
		response, err := mutate(*review.Request)
		if err != nil {
			response = &admissionv1.AdmissionResponse{}
			response.Result = &metav1.Status{Message: err.Error()}
		}
		response.UID = review.Request.UID

		resp, err := json.Marshal(&admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{
				APIVersion: admissionv1.SchemeGroupVersion.String(),
				Kind:       "AdmissionReview",
			},
			Response: response,
		})
		if err != nil {
			klog.Errorf("failed to marshal admissionResponse: %v", err)
			return
		}

		if _, err := w.Write(resp); err != nil {
			klog.Errorf("failed to write admissionResponse: %v", err)
		}
	}
}

func readReview(r *http.Request) (*admissionv1.AdmissionReview, error) {
	var body []byte
	if r.Body == nil {
		return nil, fmt.Errorf("request has no body")
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading data from request body: %v", err)
	}

	// verify the content type is accurate
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		return nil, fmt.Errorf("header Content-Type was %s, expected application/json", contentType)
	}

	admissionReview := &admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, admissionReview); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request into admissionReview: %v", err)
	}
	if admissionReview.Request == nil {
		return nil, errors.New("invalid admission review: no request defined")
	}

	return admissionReview, nil
}
