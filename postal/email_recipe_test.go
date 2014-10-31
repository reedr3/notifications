package postal_test

import (
    "encoding/json"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/fakes"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Recipes", func() {
    Describe("EmailRecipe", func() {
        var emailRecipe postal.EmailRecipe

        Describe("DispatchMail", func() {
            var fakeMailer *fakes.FakeMailer
            var fakeDBConn *fakes.FakeDBConn
            var options postal.Options
            var clientID string
            var emailID postal.EmailID
            var fakeTemplatesLoader fakes.FakeTemplatesLoader

            BeforeEach(func() {
                fakeMailer = fakes.NewFakeMailer()
                fakeTemplatesLoader = fakes.FakeTemplatesLoader{}
                emailRecipe = postal.NewEmailRecipe(fakeMailer, &fakeTemplatesLoader)

                clientID = "raptors-123"
                emailID = postal.NewEmailID()

                options = postal.Options{
                    Text: "email text",
                    To:   "dr@strangelove.com",
                }

                fakeDBConn = &fakes.FakeDBConn{}

                fakeTemplatesLoader.Templates = postal.Templates{
                    Subject: "the subject",
                    Text:    "the text",
                    HTML:    "email template",
                }
            })

            It("Calls Deliver on it's mailer with proper arguments", func() {
                emailRecipe.Dispatch(clientID, emailID, options, fakeDBConn)

                users := map[string]uaa.User{"": uaa.User{Emails: []string{options.To}}}

                Expect(len(fakeMailer.DeliverArguments)).To(Equal(7))

                Expect(fakeMailer.DeliverArguments).To(ContainElement(fakeDBConn))
                Expect(fakeMailer.DeliverArguments).To(ContainElement(fakeTemplatesLoader.Templates))
                Expect(fakeMailer.DeliverArguments).To(ContainElement(users))
                Expect(fakeMailer.DeliverArguments).To(ContainElement(options))
                Expect(fakeMailer.DeliverArguments).To(ContainElement(cf.CloudControllerSpace{}))
                Expect(fakeMailer.DeliverArguments).To(ContainElement(cf.CloudControllerOrganization{}))
                Expect(fakeMailer.DeliverArguments).To(ContainElement(clientID))
            })
        })

        Describe("Trim", func() {
            It("Trims the recipients field", func() {
                responses, err := json.Marshal([]postal.Response{
                    {
                        Status:         "delivered",
                        Email:          "user@example.com",
                        NotificationID: "123-456",
                    },
                })

                trimmedResponses := emailRecipe.Trim(responses)

                var result []map[string]string
                err = json.Unmarshal(trimmedResponses, &result)
                if err != nil {
                    panic(err)
                }

                Expect(result).To(ContainElement(map[string]string{"status": "delivered",
                    "email":           "user@example.com",
                    "notification_id": "123-456",
                }))
            })
        })
    })
})
