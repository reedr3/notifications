package web

import (
	"strings"

	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
	"github.com/cloudfoundry-incubator/notifications/web/handlers"
	"github.com/cloudfoundry-incubator/notifications/web/middleware"
	"github.com/cloudfoundry-incubator/notifications/web/services"
	"github.com/gorilla/mux"
	"github.com/ryanmoran/stack"
)

type MotherInterface interface {
	Registrar() services.Registrar
	EmailStrategy() strategies.EmailStrategy
	UserStrategy() strategies.UserStrategy
	SpaceStrategy() strategies.SpaceStrategy
	OrganizationStrategy() strategies.OrganizationStrategy
	NotificationFinder() services.NotificationFinder
	PreferencesFinder() *services.PreferencesFinder
	PreferenceUpdater() services.PreferenceUpdater
	TemplateServiceObjects() (services.TemplateFinder, services.TemplateUpdater, services.TemplateDeleter)
	Database() models.DatabaseInterface
	Logging() stack.Middleware
	ErrorWriter() handlers.ErrorWriter
	Authenticator(...string) middleware.Authenticator
	CORS() middleware.CORS
}

type Router struct {
	stacks map[string]stack.Stack
}

func NewRouter(mother MotherInterface) Router {
	registrar := mother.Registrar()
	emailStrategy := mother.EmailStrategy()
	userStrategy := mother.UserStrategy()
	spaceStrategy := mother.SpaceStrategy()
	organizationStrategy := mother.OrganizationStrategy()
	notify := handlers.NewNotify(mother.NotificationFinder(), registrar)
	preferencesFinder := mother.PreferencesFinder()
	preferenceUpdater := mother.PreferenceUpdater()
	templateFinder, templateUpdater, templateDeleter := mother.TemplateServiceObjects()
	logging := mother.Logging()
	errorWriter := mother.ErrorWriter()
	notificationsWriteAuthenticator := mother.Authenticator("notifications.write")
	notificationPreferencesReadAuthenticator := mother.Authenticator("notification_preferences.read")
	notificationPreferencesWriteAuthenticator := mother.Authenticator("notification_preferences.write")
	notificationPreferencesAdminAuthenticator := mother.Authenticator("notification_preferences.admin")
	emailsWriteAuthenticator := mother.Authenticator("emails.write")
	notificationsTemplateAdminAuthenticator := mother.Authenticator("notification_templates.admin")
	database := mother.Database()

	cors := mother.CORS()

	return Router{
		stacks: map[string]stack.Stack{
			"GET /info":                        stack.NewStack(handlers.NewGetInfo()).Use(logging),
			"POST /users/{guid}":               stack.NewStack(handlers.NewNotifyUser(notify, errorWriter, userStrategy, database)).Use(logging, notificationsWriteAuthenticator),
			"POST /spaces/{guid}":              stack.NewStack(handlers.NewNotifySpace(notify, errorWriter, spaceStrategy, database)).Use(logging, notificationsWriteAuthenticator),
			"POST /organizations/{guid}":       stack.NewStack(handlers.NewNotifyOrganization(notify, errorWriter, organizationStrategy, database)).Use(logging, notificationsWriteAuthenticator),
			"POST /emails":                     stack.NewStack(handlers.NewNotifyEmail(notify, errorWriter, emailStrategy, database)).Use(logging, emailsWriteAuthenticator),
			"PUT /registration":                stack.NewStack(handlers.NewRegisterNotifications(registrar, errorWriter, database)).Use(logging, notificationsWriteAuthenticator),
			"OPTIONS /user_preferences":        stack.NewStack(handlers.NewOptionsPreferences()).Use(logging, cors),
			"OPTIONS /user_preferences/{guid}": stack.NewStack(handlers.NewOptionsPreferences()).Use(logging, cors),
			"GET /user_preferences":            stack.NewStack(handlers.NewGetPreferences(preferencesFinder, errorWriter)).Use(logging, cors, notificationPreferencesReadAuthenticator),
			"GET /user_preferences/{guid}":     stack.NewStack(handlers.NewGetPreferencesForUser(preferencesFinder, errorWriter)).Use(logging, cors, notificationPreferencesAdminAuthenticator),
			"PATCH /user_preferences":          stack.NewStack(handlers.NewUpdatePreferences(preferenceUpdater, errorWriter, database)).Use(logging, cors, notificationPreferencesWriteAuthenticator),
			"PATCH /user_preferences/{guid}":   stack.NewStack(handlers.NewUpdateSpecificUserPreferences(preferenceUpdater, errorWriter, database)).Use(logging, cors, notificationPreferencesAdminAuthenticator),
			"GET /templates/{templateName}":    stack.NewStack(handlers.NewGetTemplates(templateFinder, errorWriter)).Use(logging, notificationsTemplateAdminAuthenticator),
			"PUT /templates/{templateName}":    stack.NewStack(handlers.NewSetTemplates(templateUpdater, errorWriter)).Use(logging, notificationsTemplateAdminAuthenticator),
			"DELETE /templates/{templateName}": stack.NewStack(handlers.NewUnsetTemplates(templateDeleter, errorWriter)).Use(logging),
		},
	}
}

func (router Router) Routes() *mux.Router {
	r := mux.NewRouter()
	for methodPath, stack := range router.stacks {
		var name = methodPath
		parts := strings.SplitN(methodPath, " ", 2)
		r.Handle(parts[1], stack).Methods(parts[0]).Name(name)
	}
	return r
}
