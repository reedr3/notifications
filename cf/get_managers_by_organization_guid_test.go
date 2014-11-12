package cf_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cloudfoundry-incubator/notifications/cf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetManagersByOrgGuid", func() {
	var testOrganizationGuid = "test-organization-guid"
	var CCServer *httptest.Server
	var ManagersEndpoint http.HandlerFunc
	var cloudController cf.CloudController

	BeforeEach(func() {
		ManagersEndpoint = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
			if token != testUAAToken {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"code":10002,"description":"Authentication error","error_code":"CF-NotAuthenticated"}`))
				return
			}

			err := req.ParseForm()
			if err != nil {
				panic(err)
			}

			organizationGuid := strings.Split(req.URL.String(), "/")[3]
			if organizationGuid != testOrganizationGuid {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"total_results":0,"total_pages":1,"prev_url":null,"next_url":null,"resources":[]}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
              "total_results": 1,
              "total_pages": 1,
              "prev_url": null,
              "next_url": null,
              "resources": [
                {
                  "metadata": {
                    "guid": "user-123",
                    "url": "/v2/users/user-123",
                    "created_at": "2013-04-30T21:00:49+00:00",
                    "updated_at": null
                  },
                  "entity": {
                    "admin": true,
                    "active": true,
                    "default_space_guid": null,
                    "spaces_url": "/v2/users/user-123/spaces",
                    "organizations_url": "/v2/users/user-123/organizations",
                    "managed_organizations_url": "/v2/users/user-123/managed_organizations",
                    "billing_managed_organizations_url": "/v2/users/user-123/billing_managed_organizations",
                    "audited_organizations_url": "/v2/users/user-123/audited_organizations",
                    "managed_spaces_url": "/v2/users/user-123/managed_spaces",
                    "audited_spaces_url": "/v2/users/user-123/audited_spaces"
                  }
                }
              ]
            }`))
		})

		CCServer = httptest.NewServer(ManagersEndpoint)
		cloudController = cf.NewCloudController(CCServer.URL, false)
	})

	AfterEach(func() {
		CCServer.Close()
	})

	It("returns a list of managers for the given organization guid", func() {
		users, err := cloudController.GetManagersByOrgGuid(testOrganizationGuid, testUAAToken)
		if err != nil {
			panic(err)
		}

		Expect(len(users)).To(Equal(1))

		Expect(users).To(ContainElement(cf.CloudControllerUser{
			GUID: "user-123",
		}))
	})

	It("returns an error when the Cloud Controller returns a 400, or 500 status code", func() {
		_, err := cloudController.GetManagersByOrgGuid(testOrganizationGuid, "bad-token")

		Expect(err).To(BeAssignableToTypeOf(cf.Failure{}))

		failure := err.(cf.Failure)
		Expect(failure.Code).To(Equal(http.StatusUnauthorized))
		Expect(failure.Message).To(Equal(`{"code":10002,"description":"Authentication error","error_code":"CF-NotAuthenticated"}`))
		Expect(failure.Error()).To(Equal(`CloudController Failure (401): {"code":10002,"description":"Authentication error","error_code":"CF-NotAuthenticated"}`))
	})

	It("returns an error when the Cloud Controller returns a 404 status code", func() {
		_, err := cloudController.GetManagersByOrgGuid("my-nonexistant-guid", testUAAToken)

		Expect(err).To(BeAssignableToTypeOf(cf.Failure{}))
		failure := err.(cf.Failure)
		Expect(failure.Code).To(Equal(http.StatusNotFound))
	})
})
