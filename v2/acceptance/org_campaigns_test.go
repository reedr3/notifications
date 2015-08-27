package acceptance

import (
	"fmt"
	"net/http"
	"strings"

	"bitbucket.org/chrj/smtpd"

	"github.com/cloudfoundry-incubator/notifications/v2/acceptance/support"
	"github.com/pivotal-cf/uaa-sso-golang/uaa"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Organization Campaigns", func() {
	var (
		client   *support.Client
		token    uaa.Token
		senderID string
	)

	BeforeEach(func() {
		client = support.NewClient(support.Config{
			Host:  Servers.Notifications.URL(),
			Trace: Trace,
		})
		token = GetClientTokenFor("my-client")

		status, response, err := client.Do("POST", "/senders", map[string]interface{}{
			"name": "my-sender",
		}, token.Access)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(http.StatusCreated))

		senderID = response["id"].(string)
	})

	It("sends a campaign to an organization", func() {
		var campaignTypeID, templateID, campaignTypeTemplateID string

		By("creating a template", func() {
			status, response, err := client.Do("POST", "/templates", map[string]interface{}{
				"name":    "Acceptance Template",
				"text":    "campaign template {{.Text}}",
				"html":    "{{.HTML}}",
				"subject": "{{.Subject}}",
			}, token.Access)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusCreated))

			templateID = response["id"].(string)
		})

		By("creating a campaign type template", func() {
			status, response, err := client.Do("POST", "/templates", map[string]interface{}{
				"name":    "CampaignType Template",
				"text":    "campaign type template {{.Text}}",
				"html":    "{{.HTML}}",
				"subject": "{{.Subject}}",
			}, token.Access)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusCreated))

			campaignTypeTemplateID = response["id"].(string)
		})

		By("creating a campaign type", func() {
			status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaign_types", senderID), map[string]interface{}{
				"name":        "some-campaign-type-name",
				"description": "acceptance campaign type",
				"template_id": campaignTypeTemplateID,
			}, token.Access)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusCreated))

			campaignTypeID = response["id"].(string)
		})

		By("sending the campaign", func() {
			status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaigns", senderID), map[string]interface{}{
				"send_to": map[string]interface{}{
					"org": "org-123",
				},
				"campaign_type_id": campaignTypeID,
				"text":             "campaign body",
				"subject":          "campaign subject",
				"template_id":      templateID,
			}, token.Access)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusAccepted))
			Expect(response["campaign_id"]).NotTo(BeEmpty())
		})

		By("seeing that the mail was delivered", func() {
			Eventually(func() []smtpd.Envelope {
				return Servers.SMTP.Deliveries
			}, "10s").Should(HaveLen(1))

			delivery := Servers.SMTP.Deliveries[0]

			Expect(delivery.Recipients).To(HaveLen(1))
			Expect(delivery.Recipients).To(ConsistOf([]string{
				"user-456@example.com",
			}))

			data := strings.Split(string(delivery.Data), "\n")
			Expect(data).To(ContainElement("campaign template campaign body"))
		})
	})

	Context("when lacking the critical scope", func() {
		It("returns a 403 forbidden", func() {
			var campaignTypeID, templateID, campaignTypeTemplateID string

			By("creating a template", func() {
				status, response, err := client.Do("POST", "/templates", map[string]interface{}{
					"name":    "Acceptance Template",
					"text":    "campaign template {{.Text}}",
					"html":    "{{.HTML}}",
					"subject": "{{.Subject}}",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				templateID = response["id"].(string)
			})

			By("creating a campaign type template", func() {
				status, response, err := client.Do("POST", "/templates", map[string]interface{}{
					"name":    "CampaignType Template",
					"text":    "campaign type template {{.Text}}",
					"html":    "{{.HTML}}",
					"subject": "{{.Subject}}",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				campaignTypeTemplateID = response["id"].(string)
			})

			By("creating a campaign type", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaign_types", senderID), map[string]interface{}{
					"name":        "some-campaign-type-name",
					"description": "acceptance campaign type",
					"template_id": campaignTypeTemplateID,
					"critical":    true,
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				campaignTypeID = response["id"].(string)
			})

			By("sending the campaign", func() {
				token = GetClientTokenFor("non-critical-client")
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaigns", senderID), map[string]interface{}{
					"send_to": map[string]interface{}{
						"org": "org-123",
					},
					"campaign_type_id": campaignTypeID,
					"text":             "campaign body",
					"subject":          "campaign subject",
					"template_id":      templateID,
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusForbidden))
				Expect(response["errors"]).To(Equal([]interface{}{"Scope critical_notifications.write is required"}))
			})
		})
	})

	Context("when the org is non-existent", func() {
		It("returns a 404 with an error message", func() {
			var campaignTypeID, templateID string

			By("creating a template", func() {
				status, response, err := client.Do("POST", "/templates", map[string]interface{}{
					"name":    "Acceptance Template",
					"text":    "{{.Text}}",
					"html":    "{{.HTML}}",
					"subject": "{{.Subject}}",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				templateID = response["id"].(string)
			})

			By("creating a campaign type", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaign_types", senderID), map[string]interface{}{
					"name":        "some-campaign-type-name",
					"description": "acceptance campaign type",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				campaignTypeID = response["id"].(string)
			})

			By("sending the campaign", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaigns", senderID), map[string]interface{}{
					"send_to": map[string]interface{}{
						"org": "missing-org",
					},
					"campaign_type_id": campaignTypeID,
					"text":             "campaign body",
					"subject":          "campaign subject",
					"template_id":      templateID,
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(response["errors"]).To(Equal([]interface{}{"The org \"missing-org\" cannot be found"}))
			})
		})
	})

	Context("when the template ID doesn't exist", func() {
		It("returns a 404 with an error message", func() {
			var campaignTypeID string

			By("creating a campaign type", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaign_types", senderID), map[string]interface{}{
					"name":        "some-campaign-type-name",
					"description": "acceptance campaign type",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				campaignTypeID = response["id"].(string)
			})

			By("sending the campaign", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaigns", senderID), map[string]interface{}{
					"send_to": map[string]interface{}{
						"org": "org-123",
					},
					"campaign_type_id": campaignTypeID,
					"text":             "campaign body",
					"subject":          "campaign subject",
					"template_id":      "missing-template-id",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(response["errors"]).To(ContainElement("Template with id \"missing-template-id\" could not be found"))
			})
		})
	})

	Context("when the campaign type ID does not exist", func() {
		It("returns a 404 with an error message", func() {
			var templateID string

			By("creating a template", func() {
				status, response, err := client.Do("POST", "/templates", map[string]interface{}{
					"name":    "Acceptance Template",
					"text":    "{{.Text}} campaign type template",
					"html":    "{{.HTML}}",
					"subject": "{{.Subject}}",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				templateID = response["id"].(string)
			})

			By("sending the campaign", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaigns", senderID), map[string]interface{}{
					"send_to": map[string]interface{}{
						"org": "org-123",
					},
					"campaign_type_id": "missing-campaign-type-id",
					"text":             "campaign body",
					"subject":          "campaign subject",
					"template_id":      templateID,
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(response["errors"]).To(ContainElement("Campaign type with id \"missing-campaign-type-id\" could not be found"))
			})
		})
	})

	Context("when omitting the template_id", func() {
		It("uses the template assigned to the campaign type", func() {
			var campaignTypeID, templateID string

			By("creating a template", func() {
				status, response, err := client.Do("POST", "/templates", map[string]interface{}{
					"name":    "Acceptance Template",
					"text":    "{{.Text}} campaign type template",
					"html":    "{{.HTML}}",
					"subject": "{{.Subject}}",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				templateID = response["id"].(string)
			})

			By("creating a campaign type", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaign_types", senderID), map[string]interface{}{
					"name":        "some-campaign-type-name",
					"description": "acceptance campaign type",
					"template_id": templateID,
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusCreated))

				campaignTypeID = response["id"].(string)
			})

			By("sending the campaign", func() {
				status, response, err := client.Do("POST", fmt.Sprintf("/senders/%s/campaigns", senderID), map[string]interface{}{
					"send_to": map[string]interface{}{
						"org": "org-123",
					},
					"campaign_type_id": campaignTypeID,
					"text":             "campaign body",
					"subject":          "campaign subject",
				}, token.Access)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusAccepted))
				Expect(response["campaign_id"]).NotTo(BeEmpty())
			})

			By("seeing that the mail was delivered", func() {
				Eventually(func() []smtpd.Envelope {
					return Servers.SMTP.Deliveries
				}, "10s").Should(HaveLen(1))

				delivery := Servers.SMTP.Deliveries[0]

				Expect(delivery.Recipients).To(HaveLen(1))
				Expect(delivery.Recipients[0]).To(Equal("user-456@example.com"))

				data := strings.Split(string(delivery.Data), "\n")
				Expect(data).To(ContainElement("campaign body campaign type template"))
			})
		})
	})
})
