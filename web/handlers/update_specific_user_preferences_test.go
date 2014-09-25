package handlers_test

import (
    "bytes"
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/cloudfoundry-incubator/notifications/models"
    "github.com/cloudfoundry-incubator/notifications/test_helpers/fakes"
    "github.com/cloudfoundry-incubator/notifications/web/handlers"
    "github.com/cloudfoundry-incubator/notifications/web/params"
    "github.com/cloudfoundry-incubator/notifications/web/services"
    "github.com/dgrijalva/jwt-go"
    "github.com/ryanmoran/stack"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("UpdateSpecificUserPreferences", func() {
    Describe("Execute", func() {

        var handler handlers.UpdateSpecificUserPreferences
        var writer *httptest.ResponseRecorder
        var request *http.Request
        var fakeDBConn *fakes.FakeDBConn
        var context stack.Context
        var updater *fakes.FakePreferenceUpdater
        var userGUID string
        var errorWriter *fakes.FakeErrorWriter

        BeforeEach(func() {

            fakeDBConn = &fakes.FakeDBConn{}
            builder := services.NewPreferencesBuilder()
            builder.Add(models.Preference{
                ClientID: "raptors",
                KindID:   "door-opening",
                Email:    false,
            })
            builder.Add(models.Preference{
                ClientID: "raptors",
                KindID:   "feeding-time",
                Email:    true,
            })
            builder.Add(models.Preference{
                ClientID: "dogs",
                KindID:   "barking",
                Email:    false,
            })
            body, err := json.MarshalIndent(builder, "", "  ")
            if err != nil {
                panic(err)
            }

            userGUID = "the-correct-user"
            request, err = http.NewRequest("PATCH", "domain/user_preferences/"+userGUID, bytes.NewBuffer(body))
            if err != nil {
                panic(err)
            }

            tokenHeader := map[string]interface{}{
                "alg": "FAST",
            }
            tokenClaims := map[string]interface{}{
                "client_id": "mister-client",
                "exp":       int64(3404281214),
            }
            rawToken := fakes.BuildToken(tokenHeader, tokenClaims)
            request.Header.Set("Authorization", "Bearer "+rawToken)

            token, err := jwt.Parse(rawToken, func(*jwt.Token) ([]byte, error) {
                return []byte(config.UAAPublicKey), nil
            })

            context = stack.NewContext()
            context.Set("token", token)

            updater = fakes.NewFakePreferenceUpdater()
            errorWriter = fakes.NewFakeErrorWriter()
            handler = handlers.NewUpdateSpecificUserPreferences(updater, errorWriter)
            writer = httptest.NewRecorder()
        })

        It("Passes the correct arguments to PreferenceUpdater Execute", func() {

            handler.Execute(writer, request, fakeDBConn, context)
            Expect(len(updater.ExecuteArguments)).To(Equal(2))

            preferencesArguments := updater.ExecuteArguments[0]

            Expect(preferencesArguments).To(ContainElement(models.Preference{
                ClientID: "raptors",
                KindID:   "door-opening",
                Email:    false,
            }))
            Expect(preferencesArguments).To(ContainElement(models.Preference{
                ClientID: "raptors",
                KindID:   "feeding-time",
                Email:    true,
            }))
            Expect(preferencesArguments).To(ContainElement(models.Preference{
                ClientID: "dogs",
                KindID:   "barking",
                Email:    false,
            }))

            Expect(updater.ExecuteArguments[1]).To(Equal(userGUID))
        })

        It("Returns a 204 status code when the Preference object does not error", func() {
            handler.Execute(writer, request, fakeDBConn, context)

            Expect(writer.Code).To(Equal(http.StatusNoContent))
        })

        Context("Failure cases", func() {
            It("delegates MissingKindOrClientErrors as params.ValidationError to the ErrorWriter", func() {
                updater.ExecuteError = services.MissingKindOrClientError("BOOM!")

                handler.Execute(writer, request, fakeDBConn, context)

                Expect(errorWriter.Error).To(Equal(params.ValidationError([]string{"BOOM!"})))

                Expect(fakeDBConn.BeginWasCalled).To(BeTrue())
                Expect(fakeDBConn.CommitWasCalled).To(BeFalse())
                Expect(fakeDBConn.RollbackWasCalled).To(BeTrue())
            })

            It("delegates CriticalKindErrors as params.ValidationError to the ErrorWriter", func() {
                updater.ExecuteError = services.CriticalKindError("BOOM!")

                handler.Execute(writer, request, fakeDBConn, context)

                Expect(errorWriter.Error).To(Equal(params.ValidationError([]string{"BOOM!"})))

                Expect(fakeDBConn.BeginWasCalled).To(BeTrue())
                Expect(fakeDBConn.CommitWasCalled).To(BeFalse())
                Expect(fakeDBConn.RollbackWasCalled).To(BeTrue())
            })
            It("delegates other errors to the ErrorWriter", func() {
                updater.ExecuteError = errors.New("BOOM!")

                handler.Execute(writer, request, fakeDBConn, context)

                Expect(errorWriter.Error).To(Equal(errors.New("BOOM!")))

                Expect(fakeDBConn.BeginWasCalled).To(BeTrue())
                Expect(fakeDBConn.CommitWasCalled).To(BeFalse())
                Expect(fakeDBConn.RollbackWasCalled).To(BeTrue())
            })

        })
    })
})