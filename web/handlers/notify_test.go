package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/notifications/application"
	"github.com/cloudfoundry-incubator/notifications/fakes"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal"
	"github.com/cloudfoundry-incubator/notifications/web/handlers"
	"github.com/cloudfoundry-incubator/notifications/web/params"
	"github.com/dgrijalva/jwt-go"
	"github.com/ryanmoran/stack"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notify", func() {
	Describe("Execute", func() {
		Context("When Emailing a user or a group", func() {
			var (
				handler       handlers.Notify
				finder        *fakes.NotificationsFinder
				validator     *fakes.Validator
				registrar     *fakes.Registrar
				request       *http.Request
				rawToken      string
				client        models.Client
				kind          models.Kind
				conn          *fakes.DBConn
				strategy      *fakes.Strategy
				context       stack.Context
				tokenHeader   map[string]interface{}
				tokenClaims   map[string]interface{}
				vcapRequestID string
				database      *fakes.Database
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
					"html":     "<p>This is the HTML Body of the email</p>",
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

				context = stack.NewContext()
				context.Set("token", token)
				context.Set("database", database)

				vcapRequestID = "some-request-id"

				conn = fakes.NewDBConn()

				handler = handlers.NewNotify(finder, registrar)
				strategy = fakes.NewStrategy()
				validator = &fakes.Validator{}
			})

			It("delegates to the mailStrategy", func() {
				_, err := handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)
				if err != nil {
					panic(err)
				}

				Expect(strategy.DispatchArguments).To(Equal([]interface{}{
					"mister-client",
					"space-001",
					"some-request-id",
					postal.Options{
						ReplyTo:           "me@example.com",
						Subject:           "Your instance is down",
						KindDescription:   "Instance Down",
						SourceDescription: "Health Monitor",
						Text:              "This is the plain text body of the email",
						HTML:              postal.HTML{BodyAttributes: "", BodyContent: "<p>This is the HTML Body of the email</p>"},
						KindID:            "test_email",
					},
				}))
			})

			It("registers the client and kind", func() {
				_, err := handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)
				Expect(err).NotTo(HaveOccurred())

				Expect(finder.ClientAndKindCall.Arguments).To(ConsistOf([]interface{}{database, "mister-client", "test_email"}))

				Expect(registrar.RegisterArguments).To(Equal([]interface{}{
					conn,
					client,
					[]models.Kind{kind},
				}))
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

						if err != nil {
							panic(err)
						}

						request, err = http.NewRequest("POST", "/spaces/space-001", bytes.NewBuffer(body))
						if err != nil {
							panic(err)
						}
						request.Header.Set("Authorization", "Bearer "+rawToken)

						_, err = handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)

						Expect(err).ToNot(BeNil())
						validationErr := err.(params.ValidationError)
						Expect(validationErr.Errors()).To(ContainElement(`boom`))
					})

					It("returns a error response when params cannot be parsed", func() {
						request, err := http.NewRequest("POST", "/spaces/space-001", strings.NewReader("this is not JSON"))
						if err != nil {
							panic(err)
						}
						request.Header.Set("Authorization", "Bearer "+rawToken)

						_, err = handler.Execute(conn, request, context, "space-001", strategy, validator, vcapRequestID)

						Expect(err).To(Equal(params.ParseError{}))
					})
				})

				Context("when the strategy dispatch method returns errors", func() {
					It("returns the error", func() {
						strategy.Error = errors.New("BOOM!")

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
						registrar.RegisterError = errors.New("BOOM!")

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
			})
		})
	})
})
