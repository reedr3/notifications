package notify_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/notifications/application"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal"
	"github.com/cloudfoundry-incubator/notifications/testing/fakes"
	"github.com/cloudfoundry-incubator/notifications/v1/services"
	"github.com/cloudfoundry-incubator/notifications/v1/web/notify"
	"github.com/cloudfoundry-incubator/notifications/web/webutil"
	"github.com/dgrijalva/jwt-go"
	"github.com/ryanmoran/stack"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notify", func() {
	Describe("Execute", func() {
		Context("When Emailing a user or a group", func() {
			var (
				handler         notify.Notify
				finder          *fakes.NotificationsFinder
				validator       *fakes.Validator
				registrar       *fakes.Registrar
				request         *http.Request
				rawToken        string
				client          models.Client
				kind            models.Kind
				conn            *fakes.Connection
				strategy        *fakes.Strategy
				context         stack.Context
				tokenHeader     map[string]interface{}
				tokenClaims     map[string]interface{}
				vcapRequestID   string
				database        *fakes.Database
				reqReceivedTime time.Time
			)

			BeforeEach(func() {
				client = models.Client{
					ID:          "mister-client",
					Description: "Health Monitor",
				}
				kind = models.Kind{
					ID:          "test_email",
					Description: "Instance Down",
					ClientID:    "mister-client",
					Critical:    true,
				}
				finder = fakes.NewNotificationsFinder()
				finder.Clients["mister-client"] = client
				finder.Kinds["test_email|mister-client"] = kind

				registrar = fakes.NewRegistrar()

				body, err := json.Marshal(map[string]string{
					"kind_id":  "test_email",
					"text":     "This is the plain text body of the email",
					"html":     "<!DOCTYPE html><html><head><script type='javascript'></script></head><body class='hello'><p>This is the HTML Body of the email</p><body></html>",
					"subject":  "Your instance is down",
					"reply_to": "me@example.com",
				})
				if err != nil {
					panic(err)
				}

				tokenHeader = map[string]interface{}{
					"alg": "FAST",
				}
				tokenClaims = map[string]interface{}{
					"client_id": "mister-client",
					"iss":       "http://zone-uaa-host/oauth/token",
					"exp":       int64(3404281214),
					"scope":     []string{"notifications.write", "critical_notifications.write"},
				}
				rawToken = fakes.BuildToken(tokenHeader, tokenClaims)

				request, err = http.NewRequest("POST", "/spaces/space-001", bytes.NewBuffer(body))
				if err != nil {
					panic(err)
				}
				request.Header.Set("Authorization", "Bearer "+rawToken)

				token, err := jwt.Parse(rawToken, func(*jwt.Token) (interface{}, error) {
					return []byte(application.UAAPublicKey), nil
				})

				database = fakes.NewDatabase()

				reqReceivedTime, _ = time.Parse(time.RFC3339Nano, "2015-06-08T14:32:11.660762586-07:00")

				context = stack.NewContext()
				context.Set("token", token)
				context.Set("database", database)
				context.Set(notify.RequestReceivedTime, reqReceivedTime)

				vcapRequestID = "some-request-id"

				conn = fakes.NewConnection()
				strategy = fakes.NewStrategy()
				validator = &fakes.Validator{}

				handler = notify.NewNotify(finder, registrar)
			})

			It("delegates to the strategy", func() {
				_, err := handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)
				Expect(err).NotTo(HaveOccurred())

				Expect(strategy.DispatchCall.Dispatch).To(Equal(services.Dispatch{
					GUID:       "space-001",
					Connection: conn,
					Client: services.DispatchClient{
						ID:          "mister-client",
						Description: "Health Monitor",
					},
					Kind: services.DispatchKind{
						ID:          "test_email",
						Description: "Instance Down",
					},
					UAAHost: "http://zone-uaa-host",
					VCAPRequest: services.DispatchVCAPRequest{
						ID:          "some-request-id",
						ReceiptTime: reqReceivedTime,
					},
					Message: services.DispatchMessage{
						ReplyTo: "me@example.com",
						Subject: "Your instance is down",
						Text:    "This is the plain text body of the email",
						HTML: services.HTML{
							BodyContent:    "<p>This is the HTML Body of the email</p>",
							BodyAttributes: `class="hello"`,
							Head:           `<script type="javascript"></script>`,
							Doctype:        "<!DOCTYPE html>",
						},
					},
				}))
			})

			It("registers the client and kind", func() {
				_, err := handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)
				Expect(err).NotTo(HaveOccurred())

				Expect(finder.ClientAndKindCall.Arguments).To(ConsistOf([]interface{}{database, "mister-client", "test_email"}))

				Expect(registrar.RegisterCall.Arguments.Connection).To(Equal(conn))
				Expect(registrar.RegisterCall.Arguments.Client).To(Equal(client))
				Expect(registrar.RegisterCall.Arguments.Kinds).To(ConsistOf([]models.Kind{kind}))
			})

			Context("failure cases", func() {
				Context("when validating params", func() {
					It("returns a error response when params are missing", func() {
						validator.ValidateErrors = []string{"boom"}

						body, err := json.Marshal(map[string]string{
							"kind_id":  "test_email",
							"text":     "This is the plain text body of the email",
							"html":     "<p>This is the HTML Body of the email</p>",
							"subject":  "Your instance is down",
							"reply_to": "me@example.com",
						})
						Expect(err).NotTo(HaveOccurred())

						request, err = http.NewRequest("POST", "/spaces/space-001", bytes.NewBuffer(body))
						Expect(err).NotTo(HaveOccurred())
						request.Header.Set("Authorization", "Bearer "+rawToken)

						_, err = handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)
						Expect(err).ToNot(BeNil())

						validationErr := err.(webutil.ValidationError)
						Expect(validationErr.Errors()).To(ContainElement(`boom`))
					})

					It("returns a error response when params cannot be parsed", func() {
						request, err := http.NewRequest("POST", "/spaces/space-001", strings.NewReader("this is not JSON"))
						Expect(err).NotTo(HaveOccurred())
						request.Header.Set("Authorization", "Bearer "+rawToken)

						_, err = handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)
						Expect(err).To(Equal(webutil.ParseError{}))
					})
				})

				Context("when the strategy dispatch method returns errors", func() {
					It("returns the error", func() {
						strategy.DispatchCall.Error = errors.New("BOOM!")

						_, err := handler.Execute(conn, request, context, "user-123", strategy, validator, vcapRequestID)
						Expect(err).To(Equal(errors.New("BOOM!")))
					})
				})

				Context("when the finder return errors", func() {
					It("returns the error", func() {
						finder.ClientAndKindCall.Error = errors.New("BOOM!")

						_, err := handler.Execute(conn, request, context, "user-123", strategy, validator, vcapRequestID)
						Expect(err).To(Equal(errors.New("BOOM!")))
					})
				})

				Context("when the registrar returns errors", func() {
					It("returns the error", func() {
						registrar.RegisterCall.Error = errors.New("BOOM!")

						_, err := handler.Execute(conn, request, context, "user-123", strategy, validator, vcapRequestID)
						Expect(err).To(Equal(errors.New("BOOM!")))
					})
				})

				Context("when trying to send a critical notification without the correct scope", func() {
					It("returns an error", func() {
						tokenClaims["scope"] = []interface{}{"notifications.write"}
						rawToken = fakes.BuildToken(tokenHeader, tokenClaims)
						token, err := jwt.Parse(rawToken, func(*jwt.Token) (interface{}, error) {
							return []byte(application.UAAPublicKey), nil
						})

						context.Set("token", token)

						_, err = handler.Execute(conn, request, context, "user-123", strategy, validator, vcapRequestID)
						Expect(err).To(BeAssignableToTypeOf(postal.NewCriticalNotificationError("test_email")))
					})
				})

				Context("when the token is mal-formed", func() {
					It("returns the error", func() {
						tokenClaims["iss"] = "%gh&%ij?"
						rawToken = fakes.BuildToken(tokenHeader, tokenClaims)
						token, err := jwt.Parse(rawToken, func(*jwt.Token) (interface{}, error) {
							return []byte(application.UAAPublicKey), nil
						})

						context.Set("token", token)

						_, err = handler.Execute(conn, request, context, "user-123", strategy, validator, vcapRequestID)
						Expect(err).To(Equal(errors.New("Token issuer URL invalid")))
					})
				})
			})
		})
	})
})
