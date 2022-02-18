/*
Copyright 2021 The Operating System Manager contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admission

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type admissionData struct {
	client    ctrlruntimeclient.Client
	namespace string
}

func New(listenAddress, namespace string, client ctrlruntimeclient.Client) (*http.Server, error) {
	mux := http.NewServeMux()
	ad := &admissionData{
		client:    client,
		namespace: namespace,
	}

	mux.HandleFunc("/machinedeployment", handleFuncFactory(ad.validateMachineDeployments))
	mux.HandleFunc("/operatingsystemprofile", handleFuncFactory(ad.validateOperatingSystemProfiles))
	mux.HandleFunc("/operatingsystemconfig", handleFuncFactory(ad.validateOperatingSystemConfigs))
	mux.HandleFunc("/healthz", healthZHandler)

	return &http.Server{
		Addr:    listenAddress,
		Handler: http.TimeoutHandler(mux, 25*time.Second, "timeout"),
	}, nil
}

func healthZHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func createAdmissionResponse(resp bool) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: resp,
	}
}

type validator func(admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)

func handleFuncFactory(validate validator) func(http.ResponseWriter, *http.Request) {
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

		// run the validation logic
		response, err := validate(*review.Request)
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
